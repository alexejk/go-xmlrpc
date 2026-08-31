//
// Package xmlrpc includes everything that is required to perform XML-RPC requests by utilizing familiar rpc.Client interface.
//
// The simplest use-case is creating a client towards an endpoint and making calls:
//
//
// 	c, _ := NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi")
//
// 	resp := &struct {
// 		BugzillaVersion struct {
// 			Version string
// 		}
// 	}{}
//
// 	err = c.Call("Bugzilla.version", nil, resp)
// 	fmt.Printf("Version: %s\n", resp.BugzillaVersion.Version)
//
//
// Additional customizations, such as setting custom headers, changing User-Agent or modifying HTTP Client used to make calls,
// pass corresponding Options to NewClient function.
//
// Servers vary in how they represent the dateTime.iso8601 type. Use the TimeFormat Option with a
// LayoutTimeFormatter (or a custom TimeFormatter) to control how time.Time values are encoded and decoded.

package xmlrpc
