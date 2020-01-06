package xmlrpc

type decodableResponse struct {
	body    []byte
	wrapper *respWrapper
}

func newDecodableResponse(body []byte) (*decodableResponse, error) {

	wrapper, err := toRespWrapper(body)
	if err != nil {
		return nil, err
	}

	r := &decodableResponse{
		wrapper: wrapper,
	}

	return r, nil
}

func (r *decodableResponse) Fault() *Fault {

	if r.wrapper.Fault == nil {
		return nil
	}

	return decodeFault(r.wrapper.Fault)
}

func (r *decodableResponse) Decode(v interface{}) error {

	return decodeWrapper(r.wrapper, v)
}
