package errors

// Integer error codes for use with handler.ErrorWithMessage().
// These mirror the codes in the AppError variables above but are provided
// as plain ints so handlers can write ErrorWithMessage(c, CodeInvalidParam, msg)
// instead of magic numbers.
const (
	CodeBadRequest     = 10000 // bad request
	CodeInvalidParam   = 10001 // invalid parameter
	CodeMissingParam   = 10002 // missing required parameter / business error
	CodeUnauthorized   = 10100 // unauthorized
	CodeForbidden      = 10200 // forbidden
	CodeTokenInvalid   = 40001 // invalid or expired token
	CodeInternal       = 50000 // internal server error
	CodeDatabase       = 50001 // database error
	CodeExternalAPI    = 50003 // external API error
)
