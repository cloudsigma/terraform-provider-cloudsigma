package cloudsigma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	cs "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"
	"github.com/cloudsigma/terraform-provider-cloudsigma/internal/cloudsigmaapi"
)

// Portable against baseline and candidate: actual native GET + actual SDK GET
// decoder and PUT encoder, rather than a hand-written serializer oracle.
var native30Fixtures = []string{`{"allow_multimount":true,"licenses":[{"amount":7,"license":{"burstable":true,"free":9007199254740993,"initial_purchase_requirement":1.2300,"subsequent_purchase_requirement":"42","long_name":"Long","name":"lic","resource_uri":"/licenses/lic/","type":"cpu","user_metric":"cores","unknown":true},"user":{"uuid":"user","resource_uri":"/users/user/"}},{"license":null,"user":null}],"unknown":{"discard":true},"media":"disk","meta":{"nested":{"keep":true},"n":9007199254740993,"fraction":1.234567890123456789,"null":null},"mounted_on":[{"uuid":"mount"}],"name":"after","owner":{"uuid":"owner"},"remote_snapshots":[{"uuid":"remote"}],"resource_uri":"/api/2.0/drives/created/","runtime":{"is_snapshotable":true,"snapshots_allocated_size":19,"storage_type":"nvme"},"size":1073741824,"snapshots":[{"uuid":"snap"}],"status":"unmounted","storage_type":"nvme","tags":[{"uuid":"old","name":"tag","meta":{"n":9007199254740993,"list":[null,{"x":true}]},"owner":{"uuid":"to","resource_uri":"/to/"},"resource_uri":"/tags/t/","resources":[{"owner":{"uuid":"ro","resource_uri":"/ro/"},"res_type":"drive","resource_uri":"/drives/r/","uuid":"r"},{}]}],"uuid":"created"}`, `{}`, `{"uuid":"d","owner":null,"runtime":null,"licenses":null,"tags":null,"meta":null}`, `{"uuid":"d","licenses":[],"tags":[],"mounted_on":[],"snapshots":[],"remote_snapshots":[],"meta":{}}`, `{"uuid":"d","licenses":[{"license":{"free":1.2300,"initial_purchase_requirement":9007199254740993,"subsequent_purchase_requirement":null}},{"license":{},"user":{}}],"tags":[{"meta":{},"resources":[],"owner":{}},{"resources":[{"owner":null}]}],"size":9007199254740993}`}

func TestNative30Projection(t *testing.T) {
	for i, fixture := range native30Fixtures {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			old := http.DefaultTransport
			defer func() { http.DefaultTransport = old }()
			http.DefaultTransport = &http.Transport{}
			conf := &Config{Username: "u", Password: "p", Location: "zrh"}
			conf.loadAndValidate(context.Background(), "fixture")
			clients, e := conf.clients()
			if e != nil {
				t.Fatal(e)
			}
			clients.native = native26Transport(t, func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fixture)), Request: r}, nil
			})
			var bodies [][]byte
			oracle := cs.NewClient(cs.NewUsernamePasswordCredentialsProvider("u", "p"), cs.WithHTTPClient(&http.Client{Transport: fakeRT(func(r *http.Request) (*http.Response, error) {
				if r.Method == "PUT" {
					b, e := io.ReadAll(r.Body)
					if e != nil {
						t.Fatal(e)
					}
					bodies = append(bodies, b)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fixture)), Request: r}, nil
			})}))
			clients.legacy = oracle
			want, _, e := oracle.Drives.Get(context.Background(), "d")
			if e != nil {
				t.Fatal(e)
			}
			got, e := driveForTagAttachment(context.Background(), clients, "d")
			if e != nil {
				t.Fatal(e)
			}
			for _, model := range []*cs.Drive{want, got} {
				_, _, e = oracle.Drives.Update(context.Background(), "d", &cs.DriveUpdateRequest{Drive: model})
				if e != nil {
					t.Fatal(e)
				}
			}
			if len(bodies) != 2 || !bytes.Equal(bodies[0], bodies[1]) {
				t.Fatalf("raw PUT mismatch %q", bodies)
			}
			t.Logf("exact SDK PUT bytes %s", bodies[1])
		})
	}
}

// Walk every populated field, including interface-held maps/slices. The walker
// changes values in place, so a shallow copy cannot satisfy the isolation oracle.
func native30Mutate(v reflect.Value) int {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			return native30Mutate(v.Elem())
		}
	case reflect.Struct:
		n := 0
		for i := 0; i < v.NumField(); i++ {
			n += native30Mutate(v.Field(i))
		}
		return n
	case reflect.Slice:
		n := 0
		for i := 0; i < v.Len(); i++ {
			n += native30Mutate(v.Index(i))
		}
		return n
	case reflect.Map:
		if v.IsNil() {
			return 0
		}
		n := 0
		for _, k := range v.MapKeys() {
			n += native30Mutate(v.MapIndex(k))
		}
		v.SetMapIndex(reflect.ValueOf("ownership-marker"), reflect.ValueOf(interface{}("changed")))
		return n + 1
	case reflect.String:
		if v.CanSet() {
			v.SetString(v.String() + "-changed")
			return 1
		}
	case reflect.Bool:
		if v.CanSet() {
			v.SetBool(!v.Bool())
			return 1
		}
	case reflect.Int:
		if v.CanSet() {
			v.SetInt(v.Int() + 1)
			return 1
		}
	case reflect.Float64:
		if v.CanSet() {
			v.SetFloat(v.Float() + 1)
			return 1
		}
	}
	return 0
}

func TestNative30Ownership(t *testing.T) {
	snapshot := func(v interface{}) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, direction := range []string{"input-to-result", "result-to-input", "repeated-result"} {
		t.Run(direction, func(t *testing.T) {
			var input cloudsigmaapi.Drive
			// String license quantities match the already-decoded native representation.
			fixture := strings.ReplaceAll(native30Fixtures[0], `"free":9007199254740993`, `"free":"9007199254740993"`)
			fixture = strings.ReplaceAll(fixture, `"initial_purchase_requirement":1.2300`, `"initial_purchase_requirement":"1.2300"`)
			if err := json.Unmarshal([]byte(fixture), &input); err != nil {
				t.Fatal(err)
			}
			result := legacyAttachmentDrive(&input)
			other := legacyAttachmentDrive(&input)
			var changed, protected interface{} = &input, result
			if direction == "result-to-input" {
				changed, protected = result, &input
			}
			if direction == "repeated-result" {
				changed, protected = result, other
			}
			beforeChanged, beforeProtected := snapshot(changed), snapshot(protected)
			if n := native30Mutate(reflect.ValueOf(changed)); n < 40 {
				t.Fatalf("vacuous mutation: %d", n)
			}
			if bytes.Equal(beforeChanged, snapshot(changed)) {
				t.Fatal("mutation did not occur")
			}
			if !bytes.Equal(beforeProtected, snapshot(protected)) {
				t.Fatal("shared ownership")
			}
		})
	}
	t.Run("negative-control", func(t *testing.T) {
		input := &cloudsigmaapi.Drive{Meta: map[string]interface{}{"nested": []interface{}{map[string]interface{}{"x": true}}}}
		shallow := *input
		before := snapshot(input)
		if native30Mutate(reflect.ValueOf(&shallow)) == 0 || bytes.Equal(before, snapshot(input)) {
			t.Fatal("oracle failed to detect alias")
		}
	})
	t.Run("null-empty-numbers", func(t *testing.T) {
		if legacyAttachmentDrive(nil) != nil {
			t.Fatal("nil drive")
		}
		nilModel := legacyAttachmentDrive(&cloudsigmaapi.Drive{})
		if nilModel.Meta != nil || nilModel.Tags != nil || nilModel.Licenses != nil || nilModel.MountedOn != nil || nilModel.Snapshots != nil || nilModel.RemoteSnapshots != nil || nilModel.Owner != nil || nilModel.Runtime != nil {
			t.Fatal("nil changed")
		}
		input := &cloudsigmaapi.Drive{Meta: map[string]interface{}{}, Tags: []cloudsigmaapi.Tag{{Meta: map[string]interface{}{}, Resources: []cloudsigmaapi.TagResource{}}}, Licenses: []cloudsigmaapi.DriveLicense{}, MountedOn: []cloudsigmaapi.ResourceLink{}, Snapshots: []cloudsigmaapi.ResourceLink{}, RemoteSnapshots: []cloudsigmaapi.ResourceLink{}}
		got := legacyAttachmentDrive(input)
		if got.Meta == nil || got.Tags == nil || got.Tags[0].Meta == nil || got.Tags[0].Resources == nil || got.Licenses == nil || got.MountedOn == nil || got.Snapshots == nil || got.RemoteSnapshots == nil {
			t.Fatal("empty changed")
		}
		if err := json.Unmarshal([]byte(`{"meta":{"n":9007199254740993,"list":[null,{},[]]}}`), input); err != nil {
			t.Fatal(err)
		}
		got = legacyAttachmentDrive(input)
		if n, ok := got.Meta["n"].(float64); !ok || n != 9007199254740992 {
			t.Fatal("legacy float64 changed")
		}
		if !reflect.DeepEqual(input.Meta, got.Meta) {
			t.Fatal("nested null/empty changed")
		}
	})
}
