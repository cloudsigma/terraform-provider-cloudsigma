package cloudsigmaapi

import (
	"bytes"
	"context"
	"encoding/json"
)

// Server represents a CloudSigma server.
type Server struct {
	// Jobs is response-only raw JSON captured by SDK service decoders.
	// It is excluded from request marshaling and direct json.Unmarshal.
	// Nil means absent; explicit null and empty arrays remain distinguishable.
	// No job schema or completion semantics are inferred.
	Jobs               json.RawMessage        `json:"-"`
	AutoStart          bool                   `json:"auto_start,omitempty"`
	Context            bool                   `json:"context,omitempty"`
	CPU                int                    `json:"cpu,omitempty"`
	CPUType            string                 `json:"cpu_type,omitempty"`
	CPUsInsteadOfCores bool                   `json:"cpus_instead_of_cores,omitempty"`
	Drives             []ServerDrive          `json:"drives,omitempty"`
	EnableNuma         bool                   `json:"enable_numa,omitempty"`
	EnclavePageCaches  []EnclavePageCache     `json:"epcs,omitempty"`
	Hypervisor         string                 `json:"hypervisor,omitempty"`
	Memory             int                    `json:"mem,omitempty"`
	Meta               map[string]interface{} `json:"meta,omitempty"`
	Name               string                 `json:"name,omitempty"`
	NICs               []ServerNIC            `json:"nics,omitempty"`
	Owner              *ResourceLink          `json:"owner,omitempty"`
	PublicKeys         []Keypair              `json:"pubkeys,omitempty"`
	ResourceURI        string                 `json:"resource_uri,omitempty"`
	Runtime            *ServerRuntime         `json:"runtime,omitempty"`
	SMP                int                    `json:"smp,omitempty"`
	Status             string                 `json:"status,omitempty"`
	Tags               []Tag                  `json:"tags,omitempty"`
	UUID               string                 `json:"uuid,omitempty"`
	VNCPassword        string                 `json:"vnc_password,omitempty"`
}

// ServerDrive represents a CloudSigma drive attached to a server.
type ServerDrive struct {
	BootOrder  int    `json:"boot_order,omitempty"`
	DevChannel string `json:"dev_channel,omitempty"`
	Device     string `json:"device,omitempty"`
	Drive      *Drive `json:"drive,omitempty"`
}

// EnclavePageCache represents a protected memory region for enclaves in a server.
type EnclavePageCache struct {
	Size int `json:"size,omitempty"`
}

// ServerNIC represents a CloudSigma network interface card attached to a server.
type ServerNIC struct {
	BootOrder        int                    `json:"boot_order,omitempty"`
	FirewallPolicy   *FirewallPolicy        `json:"firewall_policy,omitempty"`
	IP4Configuration *ServerIPConfiguration `json:"ip_v4_conf,omitempty"`
	IP6Configuration *ServerIPConfiguration `json:"ip_v6_conf,omitempty"`
	MACAddress       string                 `json:"mac,omitempty"`
	Model            string                 `json:"model,omitempty"`
	VLAN             *VLAN                  `json:"vlan,omitempty"`
}

// ServerRuntime represents a CloudSigma server runtime information.
type ServerRuntime struct {
	RuntimeNICs []ServerRuntimeNIC `json:"nics,omitempty"`
}

// ServerRuntimeNIC represents a CloudSigma server's network interface card runtime.
type ServerRuntimeNIC struct {
	InterfaceType string          `json:"interface_type,omitempty"`
	IPv4          ServerRuntimeIP `json:"ip_v4,omitempty"`
	IPv6          ServerRuntimeIP `json:"ip_v6,omitempty"`
}

// ServerRuntimeIP represents a CloudSigma server's runtime IP configuration.
type ServerRuntimeIP struct {
	ResourceURI string `json:"resource_uri,omitempty"`
	UUID        string `json:"uuid,omitempty"`
}

// ServerIPConfiguration represents a CloudSigma public IP configuration.
type ServerIPConfiguration struct {
	Type      string `json:"conf,omitempty"`
	IPAddress *IP    `json:"ip,omitempty"`
}

// FirewallPolicy represents a CloudSigma firewall policy.
type FirewallPolicy struct {
	Meta        map[string]interface{} `json:"meta,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Owner       *ResourceLink          `json:"owner,omitempty"`
	ResourceURI string                 `json:"resource_uri,omitempty"`
	Rules       []FirewallPolicyRule   `json:"rules"`
	Servers     []ResourceLink         `json:"servers"`
	Tags        []Tag                  `json:"tags"`
	UUID        string                 `json:"uuid,omitempty"`
}

// MarshalJSON is a custom marshaller for FirewallPolicy. It creates an empty
// array if Rules is nil, Servers is nil or Tags is nil.
func (f *FirewallPolicy) MarshalJSON() ([]byte, error) {
	type Alias FirewallPolicy
	a := struct{ *Alias }{(*Alias)(f)}
	if a.Rules == nil {
		a.Rules = make([]FirewallPolicyRule, 0)
	}
	if a.Servers == nil {
		a.Servers = make([]ResourceLink, 0)
	}
	if a.Tags == nil {
		a.Tags = make([]Tag, 0)
	}
	return json.Marshal(a)
}

// FirewallPolicyRule represents a CloudSigma firewall policy rule.
type FirewallPolicyRule struct {
	Action          string `json:"action,omitempty"`
	Comment         string `json:"comment,omitempty"`
	Direction       string `json:"direction,omitempty"`
	DestinationIP   string `json:"dst_ip,omitempty"`
	DestinationPort string `json:"dst_port,omitempty"`
	Protocol        string `json:"ip_proto,omitempty"`
	SourceIP        string `json:"src_ip,omitempty"`
	SourcePort      string `json:"src_port,omitempty"`
}

type serverDriveResponse ServerDrive

func (r *serverDriveResponse) UnmarshalJSON(data []byte) error {
	type alias ServerDrive
	decoded := alias(*r)
	var nested *driveResponse
	if decoded.Drive != nil {
		value := driveResponse(*decoded.Drive)
		nested = &value
	}
	wire := struct {
		*alias
		Drive *driveResponse `json:"drive"`
	}{&decoded, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Drive = (*Drive)(wire.Drive)
	*r = serverDriveResponse(decoded)
	return nil
}

type serverResponse Server

func (r *serverResponse) UnmarshalJSON(data []byte) error {
	type alias Server
	decoded := alias(*r)
	var nested []serverDriveResponse
	if decoded.Drives != nil {
		nested = make([]serverDriveResponse, len(decoded.Drives))
		for i := range nested {
			nested[i] = serverDriveResponse(decoded.Drives[i])
		}
	}
	wire := struct {
		*alias
		Jobs   json.RawMessage       `json:"jobs"`
		Drives []serverDriveResponse `json:"drives"`
	}{&decoded, nil, nested}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	decoded.Jobs = append(json.RawMessage(nil), wire.Jobs...)
	decoded.Drives = nil
	if wire.Drives != nil {
		decoded.Drives = make([]ServerDrive, len(wire.Drives))
		for i := range wire.Drives {
			decoded.Drives[i] = ServerDrive(wire.Drives[i])
		}
	}
	*r = serverResponse(decoded)
	return nil
}

// Server reads a response-only server definition; no write semantics are implied.
func (c *Client) Server(ctx context.Context, id string) (*Server, error) {
	if !driveSegment.MatchString(id) {
		return nil, &Error{Category: "transport"}
	}
	v, e := c.getLibraryDrives(ctx, serverEndpoint, "servers/"+id+"/")
	if e != nil {
		return nil, e
	}
	return v.(*Server), nil
}
func decodeServer(b []byte) (*Server, error) {
	var v serverResponse
	if e := json.NewDecoder(bytes.NewReader(b)).Decode(&v); e != nil {
		return nil, e
	}
	return (*Server)(&v), nil
}
