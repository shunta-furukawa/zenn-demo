package server

type CulcService struct{}

func NewCulcService() *CulcService {
	return &CulcService{}
}

// Add メソッド: 2つの整数を加算する
func (s *CulcService) Add(a, b int32) int32 {
	return a + b
}

// Multiply メソッド: Add を使って掛け算を模倣する
func (s *CulcService) Multiply(a, b int32) int32 {
	var result int32
	for i := int32(0); i < b; i++ {
		result = s.Add(result, a)
	}
	return result
}
