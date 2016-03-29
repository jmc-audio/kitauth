package auth

type Principal interface {
	PrincipalToken() interface{}
}

type Subject interface {
	SubjectToken() interface{}
}

type AuthNFunc func(p Principal) bool

type AuthZFunc func(p Principal, s []Subject) bool
