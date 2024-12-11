package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"project/internal/database"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/bcrypt"
)

const configFilePath = "./master_password.json"
const maxAttempts = 3 // Nombre maximal de tentatives
const lockDuration = 30 * time.Second

var masterPasswordHash string
var remainingAttempts = maxAttempts
var lockUntil time.Time

// Structure pour la configuration
type Config struct {
	MasterPasswordHash string `json:"master_password_hash"`
}

// Charger la configuration depuis un fichier
func loadConfig() (Config, error) {
	var config Config
	file, err := os.Open(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil // Pas de fichier, configuration vide
		}
		return config, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&config)
	return config, err
}

// Enregistrer la configuration dans un fichier
func saveConfig(config Config) error {
	file, err := os.OpenFile(configFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(config)
}

// Écran de verrouillage
func showLockScreen(w fyne.Window, onUnlock func()) {
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Enter Master Password"

	unlockButton := widget.NewButton("Unlock", func() {
		if time.Now().Before(lockUntil) {
			dialog.ShowError(fmt.Errorf("Too many attempts. Try again later."), w)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(masterPasswordHash), []byte(passwordEntry.Text)); err != nil {
			remainingAttempts--
			if remainingAttempts <= 0 {
				lockUntil = time.Now().Add(lockDuration)
				dialog.ShowError(fmt.Errorf("Too many attempts. Locked for %d seconds.", lockDuration/time.Second), w)
				return
			}
			dialog.ShowError(fmt.Errorf("Incorrect password. %d attempts remaining.", remainingAttempts), w)
			return
		}

		// Réinitialiser les tentatives après succès
		remainingAttempts = maxAttempts
		onUnlock()
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Master Password Required"),
		passwordEntry,
		unlockButton,
	))
}

// Configuration initiale du mot de passe maître
func setupMasterPassword(w fyne.Window, onSetup func()) {
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.PlaceHolder = "Enter New Master Password"

	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.PlaceHolder = "Confirm Master Password"

	setupButton := widget.NewButton("Set Password", func() {
		if passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("Password cannot be empty"), w)
			return
		}

		if passwordEntry.Text != confirmEntry.Text {
			dialog.ShowError(fmt.Errorf("Passwords do not match"), w)
			return
		}

		// Hacher et stocker le mot de passe maître
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordEntry.Text), bcrypt.DefaultCost)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to set password: %v", err), w)
			return
		}

		masterPasswordHash = string(hashedPassword)
		config := Config{MasterPasswordHash: masterPasswordHash}
		if err := saveConfig(config); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to save configuration: %v", err), w)
			return
		}

		onSetup()
	})

	w.SetContent(container.NewVBox(
		widget.NewLabel("Set Master Password"),
		passwordEntry,
		confirmEntry,
		setupButton,
	))
}

// Interface principale après déverrouillage
func showMainUI(w fyne.Window) {
	passwords, _ := database.GetPasswords()

	tabs := container.NewAppTabs()
	var refreshTabs func()

	refreshTabs = func() {
		if len(passwords) == 0 {
			tabs.SetItems([]*container.TabItem{
				container.NewTabItem("Add Password", createPasswordTab(&passwords, refreshTabs, w)),
			})
		} else {
			tabs.SetItems([]*container.TabItem{
				container.NewTabItem("View Passwords", viewPasswordsTab(&passwords, refreshTabs, w)),
				container.NewTabItem("Add Password", createPasswordTab(&passwords, refreshTabs, w)),
			})
		}
	}

	refreshTabs()

	lockButton := widget.NewButton("Lock", func() {
		showLockScreen(w, func() { showMainUI(w) })
	})

	w.SetContent(container.NewBorder(lockButton, nil, nil, nil, tabs))
}

// Onglet "Ajouter un mot de passe"
func createPasswordTab(passwords *[]database.Password, refreshTabs func(), w fyne.Window) fyne.CanvasObject {
	nameEntry := widget.NewEntry()
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewEntry()
	notesEntry := widget.NewMultiLineEntry()

	addButton := widget.NewButton("Add Password", func() {
		if nameEntry.Text == "" || usernameEntry.Text == "" || passwordEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("All fields except notes are required"), w)
			return
		}

		newPassword := database.Password{
			Name:     nameEntry.Text,
			Username: usernameEntry.Text,
			Password: passwordEntry.Text,
			Notes:    notesEntry.Text,
		}
		database.AddPassword(newPassword)

		*passwords, _ = database.GetPasswords()
		refreshTabs()

		nameEntry.SetText("")
		usernameEntry.SetText("")
		passwordEntry.SetText("")
		notesEntry.SetText("")
		dialog.ShowInformation("Success", "Password added successfully", w)
	})

	return container.NewVBox(
		widget.NewLabel("Add New Password"),
		widget.NewLabel("Site Name:"), nameEntry,
		widget.NewLabel("Username/Email:"), usernameEntry,
		widget.NewLabel("Password:"), passwordEntry,
		widget.NewLabel("Notes:"), notesEntry,
		addButton,
	)
}

// Onglet "Voir les mots de passe"
func viewPasswordsTab(passwords *[]database.Password, refreshTabs func(), w fyne.Window) fyne.CanvasObject {
	var selectedIndex int = -1

	passwordList := widget.NewList(
		func() int {
			return len(*passwords)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Site Name")
		},
		func(i int, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText((*passwords)[i].Name)
		},
	)

	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.Disable()
	notesEntry := widget.NewMultiLineEntry()

	copyPasswordButton := widget.NewButton("Copy Password", func() {
		if selectedIndex == -1 || selectedIndex >= len(*passwords) {
			dialog.ShowError(fmt.Errorf("No password selected"), w)
			return
		}
		w.Clipboard().SetContent(passwordEntry.Text)
		dialog.ShowInformation("Copied", "Password copied to clipboard!", w)
	})

	passwordList.OnSelected = func(id int) {
		if id < 0 || id >= len(*passwords) {
			selectedIndex = -1
			usernameEntry.SetText("")
			passwordEntry.SetText("")
			notesEntry.SetText("")
			return
		}

		selectedIndex = id
		selected := (*passwords)[id]
		usernameEntry.SetText(selected.Username)
		passwordEntry.SetText(selected.Password)
		notesEntry.SetText(selected.Notes)
	}

	return container.NewVBox(
		container.NewHSplit(passwordList, container.NewVBox(
			widget.NewLabel("Username/Email:"), usernameEntry,
			widget.NewLabel("Password:"), passwordEntry,
			widget.NewLabel("Notes:"), notesEntry,
			copyPasswordButton,
		)),
	)
}

// Configuration initiale de l'interface utilisateur
func SetupUI(w fyne.Window) {
	config, err := loadConfig()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to load configuration: %v", err), w)
		return
	}

	masterPasswordHash = config.MasterPasswordHash

	if masterPasswordHash == "" {
		setupMasterPassword(w, func() {
			showLockScreen(w, func() { showMainUI(w) })
		})
	} else {
		showLockScreen(w, func() { showMainUI(w) })
	}
}
