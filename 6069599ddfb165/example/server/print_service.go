package server

import "fmt"

type PrintService struct {
	Template string
}

func NewPrintService(templ string) *PrintService {
	return &PrintService{Template: templ}
}

// Print メソッド: 計算結果をテンプレートに埋め込んで整形
func (s *PrintService) Print(result int32) string {
	return fmt.Sprintf(s.Template, result)
}
