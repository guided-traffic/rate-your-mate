#!/bin/bash

# Wipe Script for rate-your-mate Kubernetes Environment
# Setzt die Anwendung auf Anfang zurück:
# - Löscht alle Daten aus der MySQL-Datenbank
# - Generiert ein neues JWT_SECRET
# - Restartet das Backend-Deployment
#
# Namespace: rate-your-mate
# MySQL Service: rate-your-mate-mariadb-primary
# Database: rate_your_mate

set -e

# Konfiguration
NAMESPACE="rate-your-mate"
MYSQL_SERVICE="rate-your-mate-mariadb-primary"
MYSQL_DATABASE="rate_your_mate"
SECRET_NAME="rate-your-mate-secrets"
DEPLOYMENT_NAME="rate-your-mate-backend"
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
}
trap cleanup EXIT

echo -e "${YELLOW}🧹 rate-your-mate Kubernetes Wipe Script${NC}"
echo "=========================================="
echo ""
echo -e "Namespace:  ${GREEN}$NAMESPACE${NC}"
echo -e "MySQL:      ${GREEN}$MYSQL_SERVICE${NC}"
echo -e "Database:   ${GREEN}$MYSQL_DATABASE${NC}"
echo -e "Secret:     ${GREEN}$SECRET_NAME${NC}"
echo -e "Deployment: ${GREEN}$DEPLOYMENT_NAME${NC}"
echo ""

# Sicherheitsabfrage
echo -e "${RED}⚠️  WARNUNG: Dies löscht ALLE Daten und setzt die Anwendung zurück!${NC}"
read -p "Bist du sicher? (ja/nein): " confirm
if [ "$confirm" != "ja" ]; then
    echo "Abgebrochen."
    exit 0
fi
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

# Datenbankbereinigung via lokalen MySQL-Client
echo "🗑️  Lösche Datenbank-Inhalte..."

# SQL-Befehle zum Leeren aller Tabellen (in korrekter Reihenfolge wegen Foreign Keys)
mysql -h 127.0.0.1 -P "$LOCAL_MYSQL_PORT" -u root -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE" <<EOF
SET FOREIGN_KEY_CHECKS = 0;

-- Votes löschen
TRUNCATE TABLE votes;

-- Chat-Nachrichten löschen
TRUNCATE TABLE chat_messages;

-- Gebannte User löschen
TRUNCATE TABLE banned_users;

-- Game-Cache löschen
TRUNCATE TABLE game_cache;

-- User löschen
TRUNCATE TABLE users;

SET FOREIGN_KEY_CHECKS = 1;

SELECT 'Alle Tabellen wurden geleert!' AS Status;
EOF

echo -e "${GREEN}✅ Alle Tabellen geleert${NC}"
echo ""

# Neues JWT_SECRET generieren
echo "🔐 Generiere neues JWT_SECRET..."
NEW_JWT_SECRET=$(openssl rand -base64 32)
NEW_JWT_SECRET_BASE64=$(echo -n "$NEW_JWT_SECRET" | base64)

# Secret patchen
kubectl patch secret "$SECRET_NAME" -n "$NAMESPACE" \
    --type='json' \
    -p='[{"op": "replace", "path": "/data/JWT_SECRET", "value": "'"$NEW_JWT_SECRET_BASE64"'"}]'

echo -e "${GREEN}✅ JWT_SECRET aktualisiert${NC}"
echo ""

# Backend-Deployment neustarten
echo "🔄 Restarte Backend-Deployment..."
kubectl rollout restart deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE"

# Warten auf Rollout
echo "⏳ Warte auf Rollout..."
kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout=120s

echo -e "${GREEN}✅ Backend erfolgreich neu gestartet${NC}"
echo ""

echo "=========================================="
echo -e "${GREEN}✅ Wipe abgeschlossen!${NC}"
echo ""
echo "Die Anwendung ist jetzt zurückgesetzt:"
echo "  - Alle User wurden gelöscht"
echo "  - Alle Votes wurden gelöscht"
echo "  - Alle Chat-Nachrichten wurden gelöscht"
echo "  - Alle gebannten User wurden gelöscht"
echo "  - Game-Cache wurde geleert"
echo "  - JWT_SECRET wurde erneuert (alle Sessions ungültig)"
echo "  - Backend wurde neu gestartet"
echo ""
