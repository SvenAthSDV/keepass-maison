package main

import (
	"project/internal/database"
	"project/internal/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	// Initialiser la base de données
	database.InitDB("passwords.db")

	// Initialiser l'application
	a := app.New()
	w := a.NewWindow("Password Manager")

	// Configurer l'interface utilisateur
	ui.SetupUI(w)

	// Lancer l'application
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}
