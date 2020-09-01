package jws

type ClaimSet map[string]interface{}

func (s ClaimSet) Sub() string {
	v, _ := s["sub"].(string)
	return v
}

func (s ClaimSet) Iss() string {
	v, _ := s["iss"].(string)
	return v
}

func (s ClaimSet) Groups() []string {
	v, _ := s["groups"].([]string)
	return v
}
