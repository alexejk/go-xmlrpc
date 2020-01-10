/*
Package xmlrpc includes everything that is required to perform XML-RPC requests by utilizing familiar rpc.Client interface.

The simplest use-case is creating a client towards an endpoint and making calls:


	c, _ := NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi")

	resp := &struct {
		BugzillaVersion struct {
			Version string
		}
	}{}

	err = c.Call("Bugzilla.version", nil, resp)
	fmt.Printf("Version: %s\n", resp.BugzillaVersion.Version)

*/
package xmlrpc
