#!/bin/bash

# Seed Fake Data Script for rate-your-mate
# Erstellt 15 Fake-Spieler mit 300 verteilten Bewertungspunkten
# Ein Spieler wird besonders oft negativ bewertet (im negativen Bereich)
#
# WICHTIG: Die Steam-IDs sind Fake-IDs (beginnend mit FAKE_),
# damit keine Steam-API-Abfragen gemacht werden.

set -e

# Pfad zur SQLite-Datenbank
DB_PATH="${1:-backend/data/rate-your-mate.db}"

# Prüfen ob die Datenbank existiert
if [ ! -f "$DB_PATH" ]; then
    echo "❌ Datenbank nicht gefunden: $DB_PATH"
    echo "   Bitte zuerst das Backend starten, damit die DB erstellt wird."
    echo ""
    echo "   Usage: $0 [pfad/zur/datenbank.db]"
    exit 1
fi

echo "🎮 rate-your-mate Fake Data Seeder"
echo "=================================="
echo ""
echo "📁 Datenbank: $DB_PATH"
echo ""

# Fake-Spieler Daten (15 Spieler)
# Format: steam_id|username|avatar_url
# Die Steam-IDs sind bewusst als FAKE_ markiert, damit keine API-Abfragen gemacht werden
PLAYERS=(
    "FAKE_76561198000000001|xXDarkLord420Xx|https://api.dicebear.com/7.x/pixel-art/svg?seed=darklord"
    "FAKE_76561198000000002|NoobSlayer9000|https://api.dicebear.com/7.x/pixel-art/svg?seed=noobslayer"
    "FAKE_76561198000000003|FragMaster2000|https://api.dicebear.com/7.x/pixel-art/svg?seed=fragmaster"
    "FAKE_76561198000000004|CampingKing|https://api.dicebear.com/7.x/pixel-art/svg?seed=campking"
    "FAKE_76561198000000005|ToxicAvenger|https://api.dicebear.com/7.x/pixel-art/svg?seed=toxic"
    "FAKE_76561198000000006|SupportMain|https://api.dicebear.com/7.x/pixel-art/svg?seed=support"
    "FAKE_76561198000000007|HeadshotHero|https://api.dicebear.com/7.x/pixel-art/svg?seed=headshot"
    "FAKE_76561198000000008|AFKAndy|https://api.dicebear.com/7.x/pixel-art/svg?seed=afkandy"
    "FAKE_76561198000000009|ClutchQueen|https://api.dicebear.com/7.x/pixel-art/svg?seed=clutch"
    "FAKE_76561198000000010|RageQuitRudi|https://api.dicebear.com/7.x/pixel-art/svg?seed=ragequit"
    "FAKE_76561198000000011|TeamKillerTom|https://api.dicebear.com/7.x/pixel-art/svg?seed=teamkiller"
    "FAKE_76561198000000012|ProGamerPete|https://api.dicebear.com/7.x/pixel-art/svg?seed=progamer"
    "FAKE_76561198000000013|CasualCarl|https://api.dicebear.com/7.x/pixel-art/svg?seed=casual"
    "FAKE_76561198000000014|StrategieStefan|https://api.dicebear.com/7.x/pixel-art/svg?seed=strategie"
    "FAKE_76561198000000015|FriendlyFranz|https://api.dicebear.com/7.x/pixel-art/svg?seed=friendly"
)

# Achievements (positive und negative)
POSITIVE_ACHIEVEMENTS=("pro-player" "teamplayer" "clutch-king" "support-hero" "stratege" "good-sport")
NEGATIVE_ACHIEVEMENTS=("noob" "camper" "rage-quitter" "toxic" "afk-king" "friendly-fire-expert")

echo "🧹 Lösche bestehende Fake-Daten..."
# Nur FAKE_ User und deren Votes löschen
sqlite3 "$DB_PATH" <<EOF
DELETE FROM votes WHERE from_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM votes WHERE to_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM chat_messages WHERE user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM users WHERE steam_id LIKE 'FAKE_%';
EOF

echo "👥 Erstelle 15 Fake-Spieler..."

# Spieler einfügen
for player in "${PLAYERS[@]}"; do
    IFS='|' read -r steam_id username avatar_url <<< "$player"
    sqlite3 "$DB_PATH" <<EOF
INSERT INTO users (steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at, created_at, updated_at)
VALUES ('$steam_id', '$username', '$avatar_url', '$avatar_url', 'https://steamcommunity.com/id/$username', 5, datetime('now'), datetime('now', '-' || abs(random() % 24) || ' hours'), datetime('now'));
EOF
done

echo "✅ 15 Spieler erstellt"
echo ""

# IDs der erstellten Spieler holen
echo "📊 Erstelle Bewertungen (300 Punkte)..."

# Temporäre Datei für SQL-Befehle
VOTES_SQL=$(mktemp)

# Funktion zum Generieren zufälliger Votes
generate_votes() {
    local total_points=0
    local target_points=300

    # Spieler-IDs aus der Datenbank holen
    local player_ids=$(sqlite3 "$DB_PATH" "SELECT id FROM users WHERE steam_id LIKE 'FAKE_%' ORDER BY id;")
    local ids_array=($player_ids)
    local num_players=${#ids_array[@]}

    # ToxicAvenger (Index 4, ID offset) soll viele negative Bewertungen bekommen
    local toxic_player_idx=4
    local toxic_player_id=${ids_array[$toxic_player_idx]}

    # TeamKillerTom (Index 10) soll auch einige negative bekommen
    local teamkiller_idx=10
    local teamkiller_id=${ids_array[$teamkiller_idx]}

    echo "BEGIN TRANSACTION;" >> "$VOTES_SQL"

    # Zuerst: Viele negative Votes für ToxicAvenger (ca. 25 negative = -25 Punkte)
    for i in {1..25}; do
        local from_idx=$((RANDOM % num_players))
        local from_id=${ids_array[$from_idx]}

        # Nicht selbst voten
        while [ "$from_id" == "$toxic_player_id" ]; do
            from_idx=$((RANDOM % num_players))
            from_id=${ids_array[$from_idx]}
        done

        local neg_achievement=${NEGATIVE_ACHIEVEMENTS[$((RANDOM % ${#NEGATIVE_ACHIEVEMENTS[@]}))]}
        local random_hours=$((RANDOM % 48))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $toxic_player_id, '$neg_achievement', -1, datetime('now', '-$random_hours hours', '-' || abs(random() % 60) || ' minutes'));" >> "$VOTES_SQL"
        ((total_points--))
    done

    # TeamKillerTom bekommt auch negative Votes (ca. 15)
    for i in {1..15}; do
        local from_idx=$((RANDOM % num_players))
        local from_id=${ids_array[$from_idx]}

        while [ "$from_id" == "$teamkiller_id" ]; do
            from_idx=$((RANDOM % num_players))
            from_id=${ids_array[$from_idx]}
        done

        local neg_achievement=${NEGATIVE_ACHIEVEMENTS[$((RANDOM % ${#NEGATIVE_ACHIEVEMENTS[@]}))]}
        local random_hours=$((RANDOM % 48))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $teamkiller_id, '$neg_achievement', -1, datetime('now', '-$random_hours hours', '-' || abs(random() % 60) || ' minutes'));" >> "$VOTES_SQL"
        ((total_points--))
    done

    # ProGamerPete bekommt viele positive Votes (Leader)
    local progamer_idx=11
    local progamer_id=${ids_array[$progamer_idx]}
    for i in {1..30}; do
        local from_idx=$((RANDOM % num_players))
        local from_id=${ids_array[$from_idx]}

        while [ "$from_id" == "$progamer_id" ]; do
            from_idx=$((RANDOM % num_players))
            from_id=${ids_array[$from_idx]}
        done

        local pos_achievement=${POSITIVE_ACHIEVEMENTS[$((RANDOM % ${#POSITIVE_ACHIEVEMENTS[@]}))]}
        local random_hours=$((RANDOM % 48))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $progamer_id, '$pos_achievement', 1, datetime('now', '-$random_hours hours', '-' || abs(random() % 60) || ' minutes'));" >> "$VOTES_SQL"
        ((total_points++))
    done

    # ClutchQueen und SupportMain bekommen auch viele positive
    local clutch_idx=8
    local clutch_id=${ids_array[$clutch_idx]}
    local support_idx=5
    local support_id=${ids_array[$support_idx]}

    for target_id in $clutch_id $support_id; do
        for i in {1..20}; do
            local from_idx=$((RANDOM % num_players))
            local from_id=${ids_array[$from_idx]}

            while [ "$from_id" == "$target_id" ]; do
                from_idx=$((RANDOM % num_players))
                from_id=${ids_array[$from_idx]}
            done

            local pos_achievement=${POSITIVE_ACHIEVEMENTS[$((RANDOM % ${#POSITIVE_ACHIEVEMENTS[@]}))]}
            local random_hours=$((RANDOM % 48))

            echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $target_id, '$pos_achievement', 1, datetime('now', '-$random_hours hours', '-' || abs(random() % 60) || ' minutes'));" >> "$VOTES_SQL"
            ((total_points++))
        done
    done

    # Restliche Votes zufällig verteilen bis wir 300 erreichen
    # Wir haben: -25 (toxic) -15 (teamkiller) +30 (progamer) +40 (clutch+support) = +30
    # Brauchen noch: 300 - 30 = 270 positive Punkte
    local remaining=$((target_points - total_points))

    while [ $remaining -gt 0 ]; do
        local from_idx=$((RANDOM % num_players))
        local to_idx=$((RANDOM % num_players))

        # Nicht selbst voten
        while [ "$from_idx" == "$to_idx" ]; do
            to_idx=$((RANDOM % num_players))
        done

        local from_id=${ids_array[$from_idx]}
        local to_id=${ids_array[$to_idx]}

        # 80% positive, 20% negative
        local is_positive=$((RANDOM % 100))
        local achievement=""
        local points=0

        if [ $is_positive -lt 80 ]; then
            achievement=${POSITIVE_ACHIEVEMENTS[$((RANDOM % ${#POSITIVE_ACHIEVEMENTS[@]}))]}
            points=1
        else
            achievement=${NEGATIVE_ACHIEVEMENTS[$((RANDOM % ${#NEGATIVE_ACHIEVEMENTS[@]}))]}
            points=-1
        fi

        local random_hours=$((RANDOM % 72))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $to_id, '$achievement', $points, datetime('now', '-$random_hours hours', '-' || abs(random() % 60) || ' minutes'));" >> "$VOTES_SQL"
        ((remaining -= points))
    done

    echo "COMMIT;" >> "$VOTES_SQL"
}

generate_votes

# SQL ausführen
sqlite3 "$DB_PATH" < "$VOTES_SQL"
rm "$VOTES_SQL"

echo ""
echo "📈 Statistiken:"
echo "---------------"

# Statistiken ausgeben
echo ""
echo "👥 Spieler (nach Punkten sortiert):"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT
    u.username,
    COALESCE(SUM(v.points), 0) as total_points,
    COUNT(v.id) as vote_count
FROM users u
LEFT JOIN votes v ON u.id = v.to_user_id
WHERE u.steam_id LIKE 'FAKE_%'
GROUP BY u.id
ORDER BY total_points DESC;
EOF

echo ""
echo "🏆 Achievements-Verteilung:"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT
    achievement_id,
    COUNT(*) as count,
    SUM(points) as total_points
FROM votes v
JOIN users u ON v.to_user_id = u.id
WHERE u.steam_id LIKE 'FAKE_%'
GROUP BY achievement_id
ORDER BY count DESC;
EOF

echo ""
echo "📊 Gesamtstatistik:"
sqlite3 "$DB_PATH" <<EOF
SELECT
    'Spieler: ' || COUNT(DISTINCT id) as stat
FROM users WHERE steam_id LIKE 'FAKE_%'
UNION ALL
SELECT
    'Votes: ' || COUNT(*) as stat
FROM votes v
JOIN users u1 ON v.from_user_id = u1.id
JOIN users u2 ON v.to_user_id = u2.id
WHERE u1.steam_id LIKE 'FAKE_%' AND u2.steam_id LIKE 'FAKE_%'
UNION ALL
SELECT
    'Gesamt-Punkte: ' || COALESCE(SUM(points), 0) as stat
FROM votes v
JOIN users u ON v.to_user_id = u.id
WHERE u.steam_id LIKE 'FAKE_%';
EOF

echo ""
echo "✅ Fertig! Fake-Daten wurden erfolgreich erstellt."
echo ""
echo "💡 Tipp: Die Fake-Spieler haben Steam-IDs die mit 'FAKE_' beginnen."
echo "   Diese werden nicht bei Steam abgefragt."
echo ""
echo "🗑️  Zum Löschen der Fake-Daten:"
echo "   sqlite3 $DB_PATH \"DELETE FROM votes WHERE from_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%'); DELETE FROM votes WHERE to_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%'); DELETE FROM users WHERE steam_id LIKE 'FAKE_%';\""
