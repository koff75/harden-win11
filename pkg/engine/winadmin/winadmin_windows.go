//go:build windows

package winadmin

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// isElevatedWindows utilise GetTokenInformation(TokenElevation) pour détecter
// si le process courant tourne avec un token élevé.
//
// Précédemment, on utilisait une heuristique "puis-je écrire dans C:\Windows\Temp"
// — mais sur certaines configs Win11, ce dossier est writable par les Users
// standards, ce qui produisait un faux-positif admin (bug critique : la GUI
// activait Apply/Undo pour des sessions non-élevées qui plantaient ensuite à
// la 1ère écriture HKLM).
//
// La nouvelle implémentation interroge le TokenElevation du process courant,
// qui est l'API Windows officielle (cf. UAC documentation).
func isElevatedWindows() (bool, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_QUERY,
		&token,
	); err != nil {
		return false, fmt.Errorf("OpenProcessToken: %w", err)
	}
	defer token.Close()

	var elevation uint32
	var returnedLen uint32
	// unsafe.Pointer requis par la signature Windows API GetTokenInformation
	// (cf. golang.org/x/sys/windows). Pas une vulnérabilité — c'est le bon
	// idiome pour passer un buffer sized à un syscall.
	// #nosec G103
	if err := windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&returnedLen,
	); err != nil {
		return false, fmt.Errorf("GetTokenInformation(TokenElevation): %w", err)
	}
	return elevation != 0, nil
}
