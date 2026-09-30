package cloud

type RequestOption func(*Request)

func WithApiUrl(url string) RequestOption {
	return func(r *Request) {
		r.ApiUrl = url
	}
}

func WithAuthUrl(url string) RequestOption {
	return func(r *Request) {
		r.AuthUrl = url
	}
}

func WithToken(token string) RequestOption {
	return func(r *Request) {
		r.Token = token
	}
}

func WithOrg(org string) RequestOption {
	return func(r *Request) {
		r.Org = org
	}
}

func WithSpace(space string) RequestOption {
	return func(r *Request) {
		r.Space = space
	}
}
