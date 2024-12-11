package database

type Password struct {
	ID       int
	Name     string
	Username string
	Password string
	Notes    string
}

func GetPasswords() ([]Password, error) {
	rows, err := DB.Query("SELECT id, name, username, password, notes FROM passwords")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passwords []Password
	for rows.Next() {
		var p Password
		if err := rows.Scan(&p.ID, &p.Name, &p.Username, &p.Password, &p.Notes); err != nil {
			return nil, err
		}
		passwords = append(passwords, p)
	}
	return passwords, nil
}

func AddPassword(p Password) error {
	_, err := DB.Exec("INSERT INTO passwords (name, username, password, notes) VALUES (?, ?, ?, ?)",
		p.Name, p.Username, p.Password, p.Notes)
	return err
}

func UpdatePassword(p Password) error {
	query := `
	UPDATE passwords
	SET username = ?, password = ?, notes = ?
	WHERE id = ?;
	`
	_, err := DB.Exec(query, p.Username, p.Password, p.Notes, p.ID)
	return err
}

func DeletePassword(id int) error {
	query := `
	DELETE FROM passwords
	WHERE id = ?;
	`
	_, err := DB.Exec(query, id)
	return err
}
