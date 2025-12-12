#!/bin/bash

# Seed Fake Data Script for rate-your-mate (Kubernetes Version)
# Erstellt 15 Fake-Spieler mit 300 verteilten Bewertungspunkten
# Ein Spieler wird besonders oft negativ bewertet (im negativen Bereich)
#
# WICHTIG: Die Steam-IDs sind Fake-IDs (beginnend mit FAKE_),
# damit keine Steam-API-Abfragen gemacht werden.

set -e

# Konfiguration
NAMESPACE="rate-your-mate"
MYSQL_SERVICE="rate-your-mate-mariadb-primary"
MYSQL_DATABASE="rate_your_mate"
LOCAL_MYSQL_PORT=33306

# Farben für Output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Cleanup-Funktion für Port-Forward
cleanup() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        echo ""
        echo "🔌 Beende Port-Forward..."
        kill $PORT_FORWARD_PID 2>/dev/null || true
    fi
    # Temporäre Dateien löschen
    [ -f "$VOTES_SQL" ] && rm -f "$VOTES_SQL"
}
trap cleanup EXIT

echo -e "${YELLOW}🎮 rate-your-mate Fake Data Seeder (Kubernetes)${NC}"
echo "================================================="
echo ""
echo -e "Namespace:  ${GREEN}$NAMESPACE${NC}"
echo -e "MySQL:      ${GREEN}$MYSQL_SERVICE${NC}"
echo -e "Database:   ${GREEN}$MYSQL_DATABASE${NC}"
echo ""

# Prüfen ob kubectl verfügbar ist
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}❌ kubectl nicht gefunden. Bitte installieren.${NC}"
    exit 1
fi

# Prüfen ob mysql client verfügbar ist
if ! command -v mysql &> /dev/null; then
    echo -e "${RED}❌ mysql client nicht gefunden. Bitte installieren (z.B. 'brew install mysql-client').${NC}"
    exit 1
fi

# Prüfen ob der Namespace existiert
if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo -e "${RED}❌ Namespace '$NAMESPACE' nicht gefunden.${NC}"
    exit 1
fi

# MySQL Root-Passwort aus Secret holen
echo "🔑 Hole MySQL-Credentials..."
MYSQL_ROOT_PASSWORD=$(kubectl get secret -n "$NAMESPACE" rate-your-mate-mariadb-root-credential -o jsonpath='{.data.password}' 2>/dev/null | base64 -d)

if [ -z "$MYSQL_ROOT_PASSWORD" ]; then
    echo -e "${RED}❌ MySQL Root-Passwort nicht gefunden in Secret 'rate-your-mate-mariadb-root-credential'${NC}"
    exit 1
fi
echo -e "${GREEN}✅ MySQL-Credentials gefunden${NC}"
echo ""

# Port-Forward starten
echo "🔌 Starte Port-Forward zu MySQL..."
kubectl port-forward -n "$NAMESPACE" svc/"$MYSQL_SERVICE" "$LOCAL_MYSQL_PORT":3306 &
PORT_FORWARD_PID=$!

# Warten bis Port-Forward bereit ist
sleep 3

# Prüfen ob Port-Forward läuft
if ! kill -0 $PORT_FORWARD_PID 2>/dev/null; then
    echo -e "${RED}❌ Port-Forward konnte nicht gestartet werden${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Port-Forward aktiv (localhost:$LOCAL_MYSQL_PORT)${NC}"
echo ""

# MySQL-Befehl Wrapper
mysql_cmd() {
    mysql -h 127.0.0.1 -P "$LOCAL_MYSQL_PORT" -u root -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" "$@"
}

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
mysql_cmd <<EOF
DELETE FROM votes WHERE from_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM votes WHERE to_user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM chat_messages WHERE user_id IN (SELECT id FROM users WHERE steam_id LIKE 'FAKE_%');
DELETE FROM users WHERE steam_id LIKE 'FAKE_%';
EOF
echo -e "${GREEN}✅ Bestehende Fake-Daten gelöscht${NC}"
echo ""

echo "👥 Erstelle 15 Fake-Spieler..."

# Spieler einfügen
for player in "${PLAYERS[@]}"; do
    IFS='|' read -r steam_id username avatar_url <<< "$player"
    mysql_cmd <<EOF
INSERT INTO users (steam_id, username, avatar_url, avatar_small, profile_url, credits, last_credit_at, created_at, updated_at)
VALUES ('$steam_id', '$username', '$avatar_url', '$avatar_url', 'https://steamcommunity.com/id/$username', 5, NOW(), DATE_SUB(NOW(), INTERVAL FLOOR(RAND() * 24) HOUR), NOW());
EOF
done

echo -e "${GREEN}✅ 15 Spieler erstellt${NC}"
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
    local player_ids=$(mysql_cmd -N -e "SELECT id FROM users WHERE steam_id LIKE 'FAKE_%' ORDER BY id;")
    local ids_array=($player_ids)
    local num_players=${#ids_array[@]}

    # ToxicAvenger (Index 4, ID offset) soll viele negative Bewertungen bekommen
    local toxic_player_idx=4
    local toxic_player_id=${ids_array[$toxic_player_idx]}

    # TeamKillerTom (Index 10) soll auch einige negative bekommen
    local teamkiller_idx=10
    local teamkiller_id=${ids_array[$teamkiller_idx]}

    echo "START TRANSACTION;" >> "$VOTES_SQL"

    # Zuerst: Viele negative Votes für ToxicAvenger (ca. 25 negative = -25 Punkte netto)
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
        local random_minutes=$((RANDOM % 60))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $toxic_player_id, '$neg_achievement', 1, DATE_SUB(NOW(), INTERVAL $random_hours HOUR) - INTERVAL $random_minutes MINUTE);" >> "$VOTES_SQL"
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
        local random_minutes=$((RANDOM % 60))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $teamkiller_id, '$neg_achievement', 1, DATE_SUB(NOW(), INTERVAL $random_hours HOUR) - INTERVAL $random_minutes MINUTE);" >> "$VOTES_SQL"
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
        local random_minutes=$((RANDOM % 60))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $progamer_id, '$pos_achievement', 1, DATE_SUB(NOW(), INTERVAL $random_hours HOUR) - INTERVAL $random_minutes MINUTE);" >> "$VOTES_SQL"
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
            local random_minutes=$((RANDOM % 60))

            echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $target_id, '$pos_achievement', 1, DATE_SUB(NOW(), INTERVAL $random_hours HOUR) - INTERVAL $random_minutes MINUTE);" >> "$VOTES_SQL"
            ((total_points++))
        done
    done

    # Restliche Votes zufällig verteilen bis wir 300 erreichen
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
        local net_effect=0

        if [ $is_positive -lt 80 ]; then
            achievement=${POSITIVE_ACHIEVEMENTS[$((RANDOM % ${#POSITIVE_ACHIEVEMENTS[@]}))]}
            net_effect=1
        else
            achievement=${NEGATIVE_ACHIEVEMENTS[$((RANDOM % ${#NEGATIVE_ACHIEVEMENTS[@]}))]}
            net_effect=-1
        fi

        local random_hours=$((RANDOM % 72))
        local random_minutes=$((RANDOM % 60))

        echo "INSERT INTO votes (from_user_id, to_user_id, achievement_id, points, created_at) VALUES ($from_id, $to_id, '$achievement', 1, DATE_SUB(NOW(), INTERVAL $random_hours HOUR) - INTERVAL $random_minutes MINUTE);" >> "$VOTES_SQL"
        ((remaining -= net_effect))
    done

    echo "COMMIT;" >> "$VOTES_SQL"
}

generate_votes

# SQL ausführen
mysql_cmd < "$VOTES_SQL"

echo -e "${GREEN}✅ Bewertungen erstellt${NC}"
echo ""

echo "📈 Statistiken:"
echo "---------------"

# Statistiken ausgeben
echo ""
echo "👥 Spieler (nach NETTO-Punkten sortiert):"
mysql_cmd -t <<EOF
SELECT
    u.username,
    COALESCE(SUM(
        CASE
            WHEN v.achievement_id IN ('pro-player','teamplayer','clutch-king','support-hero','stratege','good-sport')
            THEN v.points
            ELSE -v.points
        END
    ), 0) as net_points,
    COUNT(v.id) as vote_count
FROM users u
LEFT JOIN votes v ON u.id = v.to_user_id
WHERE u.steam_id LIKE 'FAKE_%'
GROUP BY u.id, u.username
ORDER BY net_points DESC;
EOF

echo ""
echo "🏆 Achievements-Verteilung:"
mysql_cmd -t <<EOF
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
mysql_cmd -t <<EOF
SELECT
    (SELECT COUNT(DISTINCT id) FROM users WHERE steam_id LIKE 'FAKE_%') as 'Spieler',
    (SELECT COUNT(*) FROM votes v
     JOIN users u1 ON v.from_user_id = u1.id
     JOIN users u2 ON v.to_user_id = u2.id
     WHERE u1.steam_id LIKE 'FAKE_%' AND u2.steam_id LIKE 'FAKE_%') as 'Votes',
    (SELECT COALESCE(SUM(points), 0) FROM votes v
     JOIN users u ON v.to_user_id = u.id
     WHERE u.steam_id LIKE 'FAKE_%') as 'Gesamt-Punkte';
EOF

echo ""
echo -e "${GREEN}✅ Fertig! Fake-Daten wurden erfolgreich erstellt.${NC}"
echo ""
echo "💡 Tipp: Die Fake-Spieler haben Steam-IDs die mit 'FAKE_' beginnen."
echo "   Diese werden nicht bei Steam abgefragt."
