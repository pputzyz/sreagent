// Package ldapx provides a lightweight LDAP client using only the Go standard
// library. It supports Bind (simple authentication) and Search operations over
// plain TCP, TLS, and StartTLS connections.
//
// This avoids an external dependency on github.com/go-ldap/ldap which pulls in
// many transitive modules. The implementation covers the subset of LDAPv3
// required for SSO authentication (bind + search + single-level deref).
package ldapx

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// ---- BER (Basic Encoding Rules) helpers ----

// berTag represents the ASN.1 BER tag class and type.
type berTag uint8

const (
	berSequence     berTag = 0x30 // SEQUENCE (constructed, universal)
	berEnumerated   berTag = 0x0A // ENUMERATED (primitive, universal)
	berOctetString  berTag = 0x04 // OCTET STRING (primitive, universal)
	berBindRequest  berTag = 0x60 // Application 0 (BindRequest)
	berBindResponse berTag = 0x61 // Application 1 (BindResponse)
	berSearchRequest berTag = 0x63 // Application 3 (SearchRequest)
	berSearchResultEntry berTag = 0x64 // Application 4 (SearchResultEntry)
	berSearchResultDone  berTag = 0x65 // Application 5 (SearchResultDone)
	berFilterEquality    berTag = 0xA3 // Context-specific, constructed (EqualityMatch)
)

// berElement is a single BER-encoded element.
type berElement struct {
	tag      berTag
	children []*berElement
	value    []byte
}

func (e *berElement) encode() []byte {
	content := e.encodeContent()
	length := encodeLength(len(content))
	header := append([]byte{byte(e.tag)}, length...)
	return append(header, content...)
}

func (e *berElement) encodeContent() []byte {
	if e.children != nil {
		var out []byte
		for _, child := range e.children {
			out = append(out, child.encode()...)
		}
		return out
	}
	return e.value
}

func encodeLength(length int) []byte {
	if length < 0x80 {
		return []byte{byte(length)}
	}
	if length <= 0xFF {
		return []byte{0x81, byte(length)}
	}
	if length <= 0xFFFF {
		return []byte{0x82, byte(length >> 8), byte(length)}
	}
	return []byte{0x83, byte(length >> 16), byte(length >> 8), byte(length)}
}

func decodeLength(data []byte, offset int) (int, int, error) {
	if offset >= len(data) {
		return 0, 0, errors.New("ldap: truncated length")
	}
	b := int(data[offset])
	offset++
	if b < 0x80 {
		return b, offset, nil
	}
	numBytes := b & 0x7F
	if offset+numBytes > len(data) {
		return 0, 0, errors.New("ldap: truncated multi-byte length")
	}
	length := 0
	for i := 0; i < numBytes; i++ {
		length = (length << 8) | int(data[offset+i])
	}
	return length, offset + numBytes, nil
}

func decodeElement(data []byte, offset int) (*berElement, int, error) {
	if offset >= len(data) {
		return nil, 0, errors.New("ldap: truncated element")
	}
	tag := berTag(data[offset])
	offset++
	length, offset, err := decodeLength(data, offset)
	if err != nil {
		return nil, 0, err
	}
	if offset+length > len(data) {
		return nil, 0, errors.New("ldap: element length exceeds data")
	}
	content := data[offset : offset+length]
	offset += length

	e := &berElement{tag: tag}

	// Check if constructed (bit 5 set)
	if tag&0x20 != 0 {
		children, err := decodeChildren(content)
		if err != nil {
			return nil, 0, err
		}
		e.children = children
	} else {
		e.value = content
	}
	return e, offset, nil
}

func decodeChildren(content []byte) ([]*berElement, error) {
	var children []*berElement
	offset := 0
	for offset < len(content) {
		child, newOffset, err := decodeElement(content, offset)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		offset = newOffset
	}
	return children, nil
}

// ---- LDAP Client ----

// Conn represents an LDAP connection.
type Conn struct {
	conn    net.Conn
	reader  io.Reader
	messageID uint32
}

// Connect establishes a TCP connection to the LDAP server.
func Connect(addr string, timeout time.Duration) (*Conn, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("ldap connect: %w", err)
	}
	return &Conn{conn: conn, reader: conn}, nil
}

// ConnectTLS establishes a TLS connection to the LDAP server (LDAPS).
func ConnectTLS(addr string, timeout time.Duration, insecureSkipVerify bool) (*Conn, error) {
	dialer := net.Dialer{Timeout: timeout}
	tlsCfg := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // user-configurable
		ServerName:         hostFromAddr(addr),
	}
	conn, err := tls.DialWithDialer(&dialer, "tcp", addr, tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("ldap tls connect: %w", err)
	}
	return &Conn{conn: conn, reader: conn}, nil
}

// StartTLS upgrades an existing plaintext connection to TLS.
func (c *Conn) StartTLS(insecureSkipVerify bool) error {
	// Send StartTLS extended request (OID 1.3.6.1.4.1.1466.20037)
	req := &berElement{
		tag: berSequence,
		children: []*berElement{
			{tag: 0x02, value: encodeInt(int(c.nextMessageID()))}, // messageID
			{
				tag: 0x77, // Application 23 (ExtendedRequest)
				children: []*berElement{
					{tag: 0x80, value: []byte("1.3.6.1.4.1.1466.20037")}, // requestName
				},
			},
		},
	}

	if _, err := c.conn.Write(req.encode()); err != nil {
		return fmt.Errorf("ldap starttls write: %w", err)
	}

	resp, err := c.readResponse()
	if err != nil {
		return fmt.Errorf("ldap starttls read: %w", err)
	}

	// Check resultCode (first child of the ExtendedResponse)
	if len(resp.children) < 2 {
		return errors.New("ldap starttls: unexpected response structure")
	}
	// The ExtendedResponse has tag 0x61 (same as BindResponse for result code parsing)
	resultCode := parseInt(resp.children[1].value)
	if resultCode != 0 {
		return fmt.Errorf("ldap starttls: server returned error code %d", resultCode)
	}

	// Upgrade to TLS
	tlsCfg := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify, //nolint:gosec // user-configurable
		ServerName:         hostFromAddr(c.conn.RemoteAddr().String()),
	}
	tlsConn := tls.Client(c.conn, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("ldap starttls handshake: %w", err)
	}
	c.conn = tlsConn
	c.reader = tlsConn
	return nil
}

// Close closes the LDAP connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

// SetTimeout sets the read/write deadline on the connection.
func (c *Conn) SetTimeout(d time.Duration) {
	_ = c.conn.SetDeadline(time.Now().Add(d))
}

func (c *Conn) nextMessageID() uint32 {
	c.messageID++
	return c.messageID
}

// ---- Bind (Simple Authentication) ----

// Bind performs a simple LDAP bind (username/password authentication).
// Returns nil on success, error on failure.
func (c *Conn) Bind(username, password string) error {
	msgID := c.nextMessageID()

	req := &berElement{
		tag: berSequence,
		children: []*berElement{
			{tag: 0x02, value: encodeInt(int(msgID))},
			{
				tag: berBindRequest,
				children: []*berElement{
					{tag: 0x02, value: encodeInt(3)}, // LDAP version 3
					{tag: berOctetString, value: []byte(username)},
					{tag: 0x80, value: []byte(password)}, // simple auth (context-specific 0)
				},
			},
		},
	}

	if _, err := c.conn.Write(req.encode()); err != nil {
		return fmt.Errorf("ldap bind write: %w", err)
	}

	resp, err := c.readResponse()
	if err != nil {
		return fmt.Errorf("ldap bind read: %w", err)
	}

	// Parse BindResponse: SEQUENCE { messageID, BindResponse { resultCode, ... } }
	if len(resp.children) < 2 {
		return errors.New("ldap bind: unexpected response structure")
	}
	bindResp := resp.children[1]
	if len(bindResp.children) < 1 {
		return errors.New("ldap bind: empty bind response")
	}
	resultCode := parseInt(bindResp.children[0].value)
	if resultCode != 0 {
		errMsg := ""
		if len(bindResp.children) >= 3 {
			errMsg = string(bindResp.children[2].value)
		}
		return fmt.Errorf("ldap bind failed (code %d): %s", resultCode, errMsg)
	}
	return nil
}

// ---- Search ----

// SearchRequest holds parameters for an LDAP search.
type SearchRequest struct {
	BaseDN       string
	Scope        int // 0=base, 1=single, 2=whole, 3=subtree
	DerefAliases int // 0=never, 1=always, 2=search, 3=base
	SizeLimit    int
	TimeLimit    int
	TypesOnly    bool
	Filter       string // e.g. "(uid=testuser)"
	Attributes   []string
}

// SearchResultEntry holds a single search result.
type SearchResultEntry struct {
	DN         string
	Attributes map[string][]string
}

// SearchResult holds the complete search result.
type SearchResult struct {
	Entries []*SearchResultEntry
}

const (
	ScopeBaseObject = 0
	ScopeSingleLevel = 1
	ScopeWholeSubtree = 2
)

// Search performs an LDAP search and returns the results.
func (c *Conn) Search(req SearchRequest) (*SearchResult, error) {
	msgID := c.nextMessageID()

	filterElement, err := parseFilter(req.Filter)
	if err != nil {
		return nil, fmt.Errorf("ldap search filter parse: %w", err)
	}

	attrElements := make([]*berElement, len(req.Attributes))
	for i, attr := range req.Attributes {
		attrElements[i] = &berElement{tag: berOctetString, value: []byte(attr)}
	}

	searchReq := &berElement{
		tag: berSequence,
		children: []*berElement{
			{tag: 0x02, value: encodeInt(int(msgID))},
			{
				tag: berSearchRequest,
				children: []*berElement{
					{tag: berOctetString, value: []byte(req.BaseDN)},
					{tag: berEnumerated, value: encodeInt(req.Scope)},
					{tag: berEnumerated, value: encodeInt(req.DerefAliases)},
					{tag: 0x02, value: encodeInt(req.SizeLimit)},
					{tag: 0x02, value: encodeInt(req.TimeLimit)},
					{tag: 0x01, value: boolBytes(req.TypesOnly)},
					filterElement,
					{tag: berSequence, children: attrElements},
				},
			},
		},
	}

	if _, err := c.conn.Write(searchReq.encode()); err != nil {
		return nil, fmt.Errorf("ldap search write: %w", err)
	}

	result := &SearchResult{}
	for {
		resp, err := c.readResponse()
		if err != nil {
			return nil, fmt.Errorf("ldap search read: %w", err)
		}

		if len(resp.children) < 2 {
			return nil, errors.New("ldap search: unexpected response structure")
		}

		tag := resp.children[1].tag
		switch tag {
		case berSearchResultEntry:
			entry, err := parseSearchResultEntry(resp.children[1])
			if err != nil {
				return nil, err
			}
			result.Entries = append(result.Entries, entry)
		case berSearchResultDone:
		resultCode := parseInt(resp.children[1].children[0].value)
			if resultCode != 0 {
				errMsg := ""
				if len(resp.children[1].children) >= 3 {
					errMsg = string(resp.children[1].children[2].value)
				}
				return nil, fmt.Errorf("ldap search failed (code %d): %s", resultCode, errMsg)
			}
			return result, nil
		default:
			// Ignore SearchResultReference and other intermediate messages
			continue
		}
	}
}

func parseSearchResultEntry(elem *berElement) (*SearchResultEntry, error) {
	if len(elem.children) < 2 {
		return nil, errors.New("ldap: search result entry has fewer than 2 children")
	}
	entry := &SearchResultEntry{
		DN:         string(elem.children[0].value),
		Attributes: make(map[string][]string),
	}

	// Parse attributes (SEQUENCE of SEQUENCEs)
	attrList := elem.children[1]
	for _, attrSeq := range attrList.children {
		if len(attrSeq.children) < 2 {
			continue
		}
		name := string(attrSeq.children[0].value)
		values := make([]string, 0, len(attrSeq.children[1].children))
		for _, valElem := range attrSeq.children[1].children {
			values = append(values, string(valElem.value))
		}
		entry.Attributes[name] = values
	}
	return entry, nil
}

// ---- Filter Parser ----

// parseFilter parses a simple LDAP filter string into a BER element.
// Supports: (attr=value), (&...), (|...), (!...)
func parseFilter(filter string) (*berElement, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" || filter == "(objectClass=*)" {
		return &berElement{tag: 0x87, value: []byte("objectClass")}, nil // present
	}

	if !strings.HasPrefix(filter, "(") || !strings.HasSuffix(filter, ")") {
		return nil, fmt.Errorf("ldap: filter must be wrapped in parentheses: %s", filter)
	}
	inner := filter[1 : len(filter)-1]

	// AND / OR / NOT
	if len(inner) > 0 && (inner[0] == '&' || inner[0] == '|' || inner[0] == '!') {
		op := inner[0]
		rest := inner[1:]

		children, err := parseFilterList(rest)
		if err != nil {
			return nil, err
		}

		var tag berTag
		switch op {
		case '&':
			tag = 0xA0 // AND
		case '|':
			tag = 0xA1 // OR
		case '!':
			tag = 0xA2 // NOT
		}
		return &berElement{tag: tag, children: children}, nil
	}

	// Simple filter: attr=value or attr~=value or attr>=value or attr<=value
	// or attr=* (present) or attr=*value* (substring - simplified)
	return parseSimpleFilter(inner)
}

func parseFilterList(s string) ([]*berElement, error) {
	var filters []*berElement
	depth := 0
	start := -1
	for i, ch := range s {
		if ch == '(' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 && start >= 0 {
				f, err := parseFilter(s[start : i+1])
				if err != nil {
					return nil, err
				}
				filters = append(filters, f)
				start = -1
			}
		}
	}
	if len(filters) == 0 {
		return nil, errors.New("ldap: empty filter list")
	}
	return filters, nil
}

func parseSimpleFilter(s string) (*berElement, error) {
	// Determine the operator
	operators := []struct {
		symbol string
		tag    berTag
	}{
		{"~=", 0xA4}, // approxMatch
		{">=", 0xA5}, // greaterOrEqual
		{"<=", 0xA6}, // lessOrEqual
		{"=", 0xA3},  // equalityMatch (must be last to avoid matching >= or <=)
	}

	for _, op := range operators {
		idx := strings.Index(s, op.symbol)
		if idx > 0 {
			attr := s[:idx]
			value := s[idx+len(op.symbol):]

			// Present filter: attr=*
			if op.tag == 0xA3 && value == "*" {
				return &berElement{tag: 0x87, value: []byte(attr)}, nil
			}

			// Substring filter (simplified: treat as equality)
			return &berElement{
				tag: op.tag,
				children: []*berElement{
					{tag: berOctetString, value: []byte(attr)},
					{tag: berOctetString, value: []byte(value)},
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("ldap: unsupported filter: %s", s)
}

// ---- Wire protocol helpers ----

func (c *Conn) readResponse() (*berElement, error) {
	// Read BER tag
	tagBuf := make([]byte, 1)
	if _, err := io.ReadFull(c.reader, tagBuf); err != nil {
		return nil, fmt.Errorf("ldap: read tag: %w", err)
	}

	// Read length
	lengthBuf := make([]byte, 1)
	if _, err := io.ReadFull(c.reader, lengthBuf); err != nil {
		return nil, fmt.Errorf("ldap: read length byte: %w", err)
	}

	var contentLen int
	b0 := lengthBuf[0]
	if b0 < 0x80 {
		contentLen = int(b0)
	} else {
		numBytes := int(b0 & 0x7F)
		lenBytes := make([]byte, numBytes)
		if _, err := io.ReadFull(c.reader, lenBytes); err != nil {
			return nil, fmt.Errorf("ldap: read multi-byte length: %w", err)
		}
		for _, b := range lenBytes {
			contentLen = (contentLen << 8) | int(b)
		}
	}

	content := make([]byte, contentLen)
	if _, err := io.ReadFull(c.reader, content); err != nil {
		return nil, fmt.Errorf("ldap: read content (%d bytes): %w", contentLen, err)
	}

	full := append(tagBuf, lengthBuf...)
	full = append(full, content...)
	_, _, _ = full[0], full[1], full[2] // suppress unused

	elem, _, err := decodeElement(append(tagBuf, append(lengthBuf, content...)...), 0)
	if err != nil {
		return nil, err
	}
	return elem, nil
}

// ---- Integer encoding helpers ----

func encodeInt(v int) []byte {
	if v == 0 {
		return []byte{0}
	}
	// Use 4 bytes for simplicity (enough for LDAP message IDs)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(v))
	// Strip leading zeros
	for len(buf) > 1 && buf[0] == 0 {
		buf = buf[1:]
	}
	// If high bit is set, prepend a zero byte (BER integer sign rule)
	if buf[0]&0x80 != 0 {
		buf = append([]byte{0}, buf...)
	}
	return buf
}

func parseInt(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	v := 0
	for _, by := range b {
		v = (v << 8) | int(by)
	}
	return v
}

func boolBytes(b bool) []byte {
	if b {
		return []byte{0xFF}
	}
	return []byte{0x00}
}

// hostFromAddr extracts hostname from "host:port" for TLS ServerName.
func hostFromAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
