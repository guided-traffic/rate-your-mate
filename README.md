# LAN Party Manager

Eine Webanwendung für LAN-Partys, bei der sich Spieler gegenseitig mit Achievements bewerten können.

## ✨ Features

- 🎮 **Steam Login** - Authentifizierung über Steam OpenID
- 💰 **Credit System** - Spieler erhalten automatisch Credits über Zeit
- 🏆 **Achievement Voting** - Spieler bewerten sich gegenseitig mit vordefinierten Achievements
- 📺 **Live Timeline** - Alle Votes in Echtzeit via WebSocket
- 🥇 **Leaderboard** - Top 3 pro Achievement

## 🚀 Installation

### Voraussetzungen

- Kubernetes Cluster
- Helm 3.x
- Steam Web API Key ([hier beantragen](https://steamcommunity.com/dev/apikey))

### Helm Repository hinzufügen

```bash
helm repo add lan-party-manager https://guided-traffic.github.io/lan-party-manager
helm repo update
```

### Installation

```bash
helm install lan-party-manager lan-party-manager/lan-party-manager \
  --set secrets.steamApiKey=DEIN_STEAM_API_KEY \
  --set secrets.jwtSecret=$(openssl rand -base64 32)
```

### Mit Ingress

```bash
helm install lan-party-manager lan-party-manager/lan-party-manager \
  --set secrets.steamApiKey=DEIN_STEAM_API_KEY \
  --set secrets.jwtSecret=$(openssl rand -base64 32) \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=lan-party.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set backend.env.FRONTEND_URL=https://lan-party.example.com \
  --set backend.env.BACKEND_URL=https://lan-party.example.com
```

### Mit eigener Values-Datei

```bash
helm install lan-party-manager lan-party-manager/lan-party-manager -f values.yaml
```

## ⚙️ Konfiguration

| Parameter | Beschreibung | Default |
|-----------|--------------|---------|
| `secrets.steamApiKey` | Steam Web API Key (erforderlich) | `""` |
| `secrets.jwtSecret` | JWT Secret für Token-Signierung (erforderlich) | `""` |
| `backend.env.CREDIT_INTERVAL_MINUTES` | Minuten zwischen Credit-Vergabe | `10` |
| `backend.env.CREDIT_MAX` | Maximale Credits pro Spieler | `10` |
| `backend.env.JWT_EXPIRATION_DAYS` | JWT Gültigkeit in Tagen | `7` |
| `ingress.enabled` | Ingress aktivieren | `false` |
| `ingress.hosts` | Ingress Hosts Konfiguration | `[]` |

Alle verfügbaren Optionen findest du in der [values.yaml](helm/lan-party-manager/values.yaml).

## 🛠️ Entwicklung

### Voraussetzungen

- Node.js 20+
- Go 1.22+

### Frontend

```bash
cd frontend
npm install
npm start
```

### Backend

```bash
cd backend
go mod tidy
go run main.go
```

## 🎨 Credits

Achievement-Icons von [Game-icons.net](https://game-icons.net) unter [CC BY 3.0](https://creativecommons.org/licenses/by/3.0/) Lizenz.

## 📄 Lizenz

MIT
