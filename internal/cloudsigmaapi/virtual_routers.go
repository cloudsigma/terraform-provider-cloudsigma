package cloudsigmaapi

import "context"

// VirtualRouters reads the virtual-router collection as dynamic JSON. The
// router configuration is deployment- and subscription-specific, so this
// library method deliberately does not model or mutate child configuration.
func (c *Client) VirtualRouters(ctx context.Context) (DynamicDocument, error) {
	v, err := c.getLibraryDrives(ctx, virtualRoutersEndpoint, "virtual_routers/?limit=0")
	if err != nil {
		return nil, err
	}
	return v.(DynamicDocument), nil
}
