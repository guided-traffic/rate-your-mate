package models

// Achievement represents a predefined achievement that users can vote for
type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	IsPositive  bool   `json:"is_positive"`
}

// Predefined achievements
var Achievements = map[string]Achievement{
	// Positive achievements
	"pro-player": {
		ID:          "pro-player",
		Name:        "Pro Player",
		Description: "Zeigt herausragende Fähigkeiten",
		ImageURL:    "/icons/achievements/trophy.svg",
		IsPositive:  true,
	},
	"endboss": {
		ID:          "endboss",
		Name:        "Endboss",
		Description: "Unbesiegbar wie ein Endboss",
		ImageURL:    "/icons/achievements/overlord-helm.svg",
		IsPositive:  true,
	},
	"teamplayer": {
		ID:          "teamplayer",
		Name:        "Teamplayer",
		Description: "Setzt das Team immer an erste Stelle",
		ImageURL:    "/icons/achievements/three-friends.svg",
		IsPositive:  true,
	},
	"mvp": {
		ID:          "mvp",
		Name:        "MVP",
		Description: "Most Valuable Player der Runde",
		ImageURL:    "/icons/achievements/laurel-crown.svg",
		IsPositive:  true,
	},
	"clutch-king": {
		ID:          "clutch-king",
		Name:        "Clutch King",
		Description: "Rettet aussichtslose Situationen",
		ImageURL:    "/icons/achievements/muscle-up.svg",
		IsPositive:  true,
	},
	"support-hero": {
		ID:          "support-hero",
		Name:        "Support Hero",
		Description: "Immer zur Stelle wenn man Hilfe braucht",
		ImageURL:    "/icons/achievements/shaking-hands.svg",
		IsPositive:  true,
	},
	"stratege": {
		ID:          "stratege",
		Name:        "Stratege",
		Description: "Plant jeden Zug wie ein Schachmeister",
		ImageURL:    "/icons/achievements/chess-king.svg",
		IsPositive:  true,
	},
	"good-sport": {
		ID:          "good-sport",
		Name:        "Gute Manieren",
		Description: "Verliert mit Würde, gewinnt mit Bescheidenheit",
		ImageURL:    "/icons/achievements/bow-tie-ribbon.svg",
		IsPositive:  true,
	},

	// Negative achievements
	"noob": {
		ID:          "noob",
		Name:        "Noob",
		Description: "Hat noch viel zu lernen",
		ImageURL:    "/icons/achievements/baby-face.svg",
		IsPositive:  false,
	},
	"camper": {
		ID:          "camper",
		Name:        "Camper",
		Description: "Bewegt sich nur wenn es unbedingt sein muss",
		ImageURL:    "/icons/achievements/hidden.svg",
		IsPositive:  false,
	},
	"rage-quitter": {
		ID:          "rage-quitter",
		Name:        "Rage Quitter",
		Description: "Verlässt das Spiel wenn es nicht läuft",
		ImageURL:    "/icons/achievements/enrage.svg",
		IsPositive:  false,
	},
	"toxic": {
		ID:          "toxic",
		Name:        "Toxic",
		Description: "Verbreitet schlechte Stimmung",
		ImageURL:    "/icons/achievements/death-juice.svg",
		IsPositive:  false,
	},
	"lagger": {
		ID:          "lagger",
		Name:        "Lagger",
		Description: "Ping ist nur eine Zahl... eine sehr hohe",
		ImageURL:    "/icons/achievements/snail.svg",
		IsPositive:  false,
	},
	"afk-king": {
		ID:          "afk-king",
		Name:        "AFK King",
		Description: "Ist öfter AFK als am Spielen",
		ImageURL:    "/icons/achievements/sleepy.svg",
		IsPositive:  false,
	},
	"friendly-fire-expert": {
		ID:          "friendly-fire-expert",
		Name:        "Friendly Fire Expert",
		Description: "Trifft Teamkameraden besser als Gegner",
		ImageURL:    "/icons/achievements/backstab.svg",
		IsPositive:  false,
	},
}

// GetAllAchievements returns all achievements as a slice
func GetAllAchievements() []Achievement {
	achievements := make([]Achievement, 0, len(Achievements))
	for _, a := range Achievements {
		achievements = append(achievements, a)
	}
	return achievements
}

// GetAchievement returns an achievement by ID
func GetAchievement(id string) (Achievement, bool) {
	a, ok := Achievements[id]
	return a, ok
}

// IsValidAchievement checks if an achievement ID is valid
func IsValidAchievement(id string) bool {
	_, ok := Achievements[id]
	return ok
}
