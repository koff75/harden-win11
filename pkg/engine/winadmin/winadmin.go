// Package winadmin détecte si le process courant tourne avec des privilèges
// administrateur sur Windows. Sur Linux/macOS, IsElevated retourne false sans
// erreur (le moteur est Windows-only mais on évite de faire planter les tests
// qui tournent sur d'autres OS).
package winadmin

import "runtime"

// IsElevated retourne true si le process courant a des privilèges admin
// (membre actif du groupe BUILTIN\Administrators avec UAC élevé).
//
// Le check est fait via une tentative d'ouverture en write d'un fichier dans
// %WINDIR% (qui requiert admin). Pas le plus élégant mais évite la dépendance
// à golang.org/x/sys/windows et les complexités du token Windows.
func IsElevated() (bool, error) {
	if runtime.GOOS != "windows" {
		return false, nil
	}
	return isElevatedWindows()
}
