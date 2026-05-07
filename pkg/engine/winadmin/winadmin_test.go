package winadmin

import "testing"

// TestIsElevated vérifie que l'appel ne plante pas. Le résultat dépend du
// contexte d'exécution (admin ou pas), donc on ne le compare pas — on vérifie
// juste qu'on a un (bool, nil) sans crash.
func TestIsElevated_DoesNotPanic(t *testing.T) {
	_, err := IsElevated()
	if err != nil {
		t.Errorf("IsElevated returned unexpected error: %v", err)
	}
}
