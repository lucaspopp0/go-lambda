package middleware

func DefaultChain[Req Request, Resp Response](next HandlerFunc[Req, Resp]) HandlerFunc[Req, Resp] {
	withCORS := WithCORS[Req, Resp](CORSOptions{
		ValidOrigins: []string{
			"http://localhost:8081",
		},
	})

	return WithLogger(
		withCORS(
			next,
		),
	)
}
