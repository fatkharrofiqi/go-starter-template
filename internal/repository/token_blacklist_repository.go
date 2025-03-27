package repository

type TokenBlacklist struct {
	// This could be a map, a database table, or any other storage mechanism.
	blacklist map[string]struct{}
}

func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{blacklist: make(map[string]struct{})}
}

func (tb *TokenBlacklist) Add(token string) error {
	tb.blacklist[token] = struct{}{}
	return nil
}

func (tb *TokenBlacklist) IsBlacklisted(token string) bool {
	_, exists := tb.blacklist[token]
	return exists
}
