//go:build !linux && !darwin

package gear

func init() {
	var _ int = "gear supports only linux and darwin"
}
