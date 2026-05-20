/**
 * RBAC Permission Constants
 *
 * Centralized permission strings matching backend buildPermissions().
 * Use with usePermissions().hasPerm() or v-can directive.
 */

// ── Rules ──
export const PERM_RULES_VIEW = 'rules.view'
export const PERM_RULES_CREATE = 'rules.create'
export const PERM_RULES_EDIT = 'rules.edit'
export const PERM_RULES_DELETE = 'rules.delete'
export const PERM_RULES_MANAGE = 'rules.manage' // create + edit + delete

// ── Events ──
export const PERM_EVENTS_VIEW = 'events.view'
export const PERM_EVENTS_ACK = 'events.ack'
export const PERM_EVENTS_ASSIGN = 'events.assign'
export const PERM_EVENTS_MANAGE = 'events.manage'

// ── Incidents ──
export const PERM_INCIDENTS_VIEW = 'incidents.view'
export const PERM_INCIDENTS_CREATE = 'incidents.create'
export const PERM_INCIDENTS_MANAGE = 'incidents.manage'

// ── Schedules ──
export const PERM_SCHEDULES_VIEW = 'schedules.view'
export const PERM_SCHEDULES_MANAGE = 'schedules.manage'

// ── Channels ──
export const PERM_CHANNELS_VIEW = 'channels.view'
export const PERM_CHANNELS_MANAGE = 'channels.manage'

// ── Data Sources ──
export const PERM_DATASOURCES_VIEW = 'datasources.view'
export const PERM_DATASOURCES_MANAGE = 'datasources.manage'

// ── Dashboards ──
export const PERM_DASHBOARDS_VIEW = 'dashboards.view'
export const PERM_DASHBOARDS_MANAGE = 'dashboards.manage'

// ── Users & Teams ──
export const PERM_USERS_MANAGE = 'users.manage'
export const PERM_TEAMS_MANAGE = 'teams.manage'
export const PERM_ROLES_VIEW = 'roles.view'

// ── Settings ──
export const PERM_SETTINGS_MANAGE = 'settings.manage'
export const PERM_AUDIT_VIEW = 'audit.view'

// ── Notifications & Todos ──
export const PERM_NOTIFICATIONS_VIEW = 'notifications.view'
export const PERM_TODOS_VIEW = 'todos.view'
export const PERM_TODOS_MANAGE = 'todos.manage'

// ── .write suffix constants (aligned with backend RBAC rbac.go, PR5-A task 1.5) ──
export const PERM_CHANNELS_WRITE = 'channels.write'
export const PERM_DATASOURCE_WRITE = 'datasource.write'
export const PERM_DISPATCH_WRITE = 'dispatch.write'
export const PERM_INHIBITION_WRITE = 'inhibition.write'
export const PERM_INTEGRATION_WRITE = 'integration.write'
export const PERM_MUTE_WRITE = 'mute.write'
export const PERM_NOTIFY_WRITE = 'notify.write'
export const PERM_RULES_WRITE = 'rules.write'
export const PERM_TEAM_WRITE = 'team.write'
export const PERM_USER_WRITE = 'user.write'

// ── Preset Groups for common UI patterns ──

/** Can create/edit/delete alert rules */
export const PERM_RULE_WRITE = [PERM_RULES_MANAGE, PERM_RULES_CREATE, PERM_RULES_EDIT, PERM_RULES_DELETE]

/** Can manage system settings (admin-only features) */
export const PERM_ADMIN_FEATURES = [PERM_SETTINGS_MANAGE, PERM_USERS_MANAGE]

/** Can manage incidents (ack, assign, escalate) */
export const PERM_INCIDENT_OPS = [PERM_INCIDENTS_MANAGE, PERM_EVENTS_ACK, PERM_EVENTS_ASSIGN]
