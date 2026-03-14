package cloud

type RequestOpt func(*Request)

func WithApiUrl(url string) RequestOpt {
	return func(r *Request) {
		r.ApiUrl = url
	}
}

func WithAuthUrl(url string) RequestOpt {
	return func(r *Request) {
		r.AuthUrl = url
	}
}

func WithToken(token string) RequestOpt {
	return func(r *Request) {
		r.Token = token
	}
}

func WithOrg(org string) RequestOpt {
	return func(r *Request) {
		r.Org = org
	}
}

func WithSpace(space string) RequestOpt {
	return func(r *Request) {
		r.Space = space
	}
}
