package cloudsigmaapi

import "context"

// NotificationContacts reads the account's notification contact collection as
// dynamic JSON. Contact fields and pagination metadata are preserved exactly.
// No contact mutation or ownership semantics are inferred here.
func (c *Client) NotificationContacts(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, notificationContactsEndpoint, "notification_contacts/?limit=0", nil)
}

// NotificationPreferences reads the account's notification preference
// collection as dynamic JSON. PUT/POST/DELETE behavior remains outside this
// read-only method because the published mutation responses differ by
// deployment.
func (c *Client) NotificationPreferences(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, notificationPreferencesEndpoint, "notification_preferences/?limit=0", nil)
}

// ACLs reads the account-owned ACL collection as dynamic JSON. ACL rules and
// grantee shapes are deployment-specific and are not used to infer Terraform
// ownership or sharing lifecycle semantics.
func (c *Client) ACLs(ctx context.Context) (DynamicDocument, error) {
	return c.dynamicDocument(ctx, aclsEndpoint, "acls/?limit=0", nil)
}
