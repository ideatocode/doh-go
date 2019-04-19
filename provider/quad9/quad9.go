/*
 * Copyright 2019 Li Kexian
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * DNS over HTTPS (DoH) Golang Implementation
 * https://www.likexian.com/
 */

package quad9

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/likexian/doh-go/dns"
	"github.com/likexian/gokit/xhttp"
	"github.com/likexian/gokit/xip"
	"golang.org/x/net/idna"
	"strings"
)

// Provider is a DoH provider client
type Provider struct {
	provides int
	xhttp    *xhttp.Request
}

const (
	// DefaultProvides is default provides, Recommended (Secure)
	DefaultProvides = iota
	// SecuredProvides Provides: Security blocklist, DNSSEC, No EDNS Client-Subnet sent
	SecuredProvides
	// Unsecured Provides: No security blocklist, no DNSSEC, No EDNS Client-Subnet sent
	UnsecuredProvides
)

var (
	// Upstream is DoH query upstream
	Upstream = map[int]string{
		DefaultProvides:   "https://9.9.9.9/dns-query",
		SecuredProvides:   "https://dns9.quad9.net/dns-query",
		UnsecuredProvides: "https://dns10.quad9.net/dns-query",
	}
)

// Version returns package version
func Version() string {
	return "0.5.0"
}

// Author returns package author
func Author() string {
	return "[Li Kexian](https://www.likexian.com/)"
}

// License returns package license
func License() string {
	return "Licensed under the Apache License 2.0"
}

// New returns a new quad9 provider client
func New() *Provider {
	return &Provider{
		provides: DefaultProvides,
		xhttp:    xhttp.New(),
	}
}

// String returns string of provider
func (c *Provider) String() string {
	return "quad9"
}

// SetProvides set upstream provides type, quad9 does NOT supported
func (c *Provider) SetProvides(p int) error {
	if _, ok := Upstream[p]; !ok {
		return fmt.Errorf("doh: quad9: not supported provides: %d", p)
	}

	c.provides = p

	return nil
}

// Query do DoH query
func (c *Provider) Query(ctx context.Context, d dns.Domain, t dns.Type) (*dns.Response, error) {
	return c.ECSQuery(ctx, d, t, "")
}

// ECSQuery do DoH query with the edns0-client-subnet option
func (c *Provider) ECSQuery(ctx context.Context, d dns.Domain, t dns.Type, s dns.ECS) (*dns.Response, error) {
	name := strings.TrimSpace(string(d))
	name, err := idna.ToASCII(name)
	if err != nil {
		return nil, err
	}

	param := xhttp.QueryParam{
		"name": name,
		"type": strings.TrimSpace(string(t)),
	}

	ss := strings.TrimSpace(string(s))
	if ss != "" {
		ss, err := xip.FixSubnet(ss)
		if err != nil {
			return nil, err
		}
		param["edns_client_subnet"] = ss
	}

	rsp, err := c.xhttp.Get(Upstream[c.provides], param, ctx, xhttp.Header{"accept": "application/dns-json"})
	if err != nil {
		return nil, err
	}

	defer rsp.Close()
	buf, err := rsp.Bytes()
	if err != nil {
		return nil, err
	}

	rr := &dns.Response{
		Provider: c.String(),
	}
	err = json.NewDecoder(bytes.NewBuffer(buf)).Decode(rr)
	if err != nil {
		return nil, err
	}

	if rr.Status != 0 {
		return rr, fmt.Errorf("doh: quad9: failed response code %d", rr.Status)
	}

	return rr, nil
}
