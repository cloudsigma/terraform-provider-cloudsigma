package cloudsigma

import (
	"context"

	"github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

// driveForTagAttachment migrates the read only. The complete pinned SDK model
// still feeds the legacy PUT; this adapter does not authorize new writable fields.
func driveForTagAttachment(ctx context.Context, meta interface{}, id string) (*cloudsigma.Drive, error) {
	client, err := extractLegacy(meta)
	if err != nil {
		return nil, err
	}
	if clients, ok := meta.(*configClients); ok && clients.native != nil {
		if clients.basicCredentialError != nil {
			return nil, clients.basicCredentialError
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		drive, err := clients.native.DriveWithCallerContext(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return legacyAttachmentDrive(drive), nil
	}
	drive, _, err := client.Drives.Get(ctx, id)
	return drive, err
}

// legacyAttachmentDrive projects the complete decoded native model at the sole
// SDK PUT boundary. It does not authorize response fields for new mutations.
// Struct conversions deliberately fail compilation if either pinned wire model
// drifts; pointer/slice/map ownership is independent of the source read.
func legacyAttachmentDrive(d *cloudsigmaapi.Drive) *cloudsigma.Drive {
	if d == nil {
		return nil
	}
	r := &cloudsigma.Drive{
		AllowMultimount: d.AllowMultimount, Media: d.Media, Meta: attachmentMeta(d.Meta),
		MountedOn: attachmentLinks(d.MountedOn), Name: d.Name, Owner: attachmentLink(d.Owner),
		RemoteSnapshots: attachmentLinks(d.RemoteSnapshots), ResourceURI: d.ResourceURI,
		Size: d.Size, Snapshots: attachmentLinks(d.Snapshots), Status: d.Status,
		StorageType: d.StorageType, UUID: d.UUID,
	}
	if d.Runtime != nil {
		v := cloudsigma.DriveRuntime(*d.Runtime)
		r.Runtime = &v
	}
	if d.Licenses != nil {
		r.Licenses = make([]cloudsigma.DriveLicense, len(d.Licenses))
		for i, v := range d.Licenses {
			r.Licenses[i] = cloudsigma.DriveLicense{Amount: v.Amount, User: attachmentLink(v.User)}
			if v.License != nil {
				l := cloudsigma.License(*v.License)
				r.Licenses[i].License = &l
			}
		}
	}
	if d.Tags != nil {
		r.Tags = make([]cloudsigma.Tag, len(d.Tags))
		for i, v := range d.Tags {
			r.Tags[i] = cloudsigma.Tag{Meta: attachmentMeta(v.Meta), Name: v.Name, Owner: attachmentLink(v.Owner), ResourceURI: v.ResourceURI, UUID: v.UUID}
			if v.Resources != nil {
				r.Tags[i].Resources = make([]cloudsigma.TagResource, len(v.Resources))
				for j, x := range v.Resources {
					r.Tags[i].Resources[j] = cloudsigma.TagResource{Owner: attachmentLink(x.Owner), ResourceType: x.ResourceType, ResourceURI: x.ResourceURI, UUID: x.UUID}
				}
			}
		}
	}
	return r
}
func attachmentLink(v *cloudsigmaapi.ResourceLink) *cloudsigma.ResourceLink {
	if v == nil {
		return nil
	}
	r := cloudsigma.ResourceLink(*v)
	return &r
}
func attachmentLinks(v []cloudsigmaapi.ResourceLink) []cloudsigma.ResourceLink {
	if v == nil {
		return nil
	}
	r := make([]cloudsigma.ResourceLink, len(v))
	for i, x := range v {
		r[i] = cloudsigma.ResourceLink(x)
	}
	return r
}

// Metadata originates in encoding/json: numbers deliberately remain float64,
// including its legacy rounding. No UseNumber or stricter validation is added.
func attachmentValue(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return attachmentMeta(x)
	case []interface{}:
		if x == nil {
			return []interface{}(nil)
		}
		r := make([]interface{}, len(x))
		for i, y := range x {
			r[i] = attachmentValue(y)
		}
		return r
	default:
		return v
	}
}
func attachmentMeta(v map[string]interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	r := make(map[string]interface{}, len(v))
	for k, x := range v {
		r[k] = attachmentValue(x)
	}
	return r
}
