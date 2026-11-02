# API de gestion de garage

Projet Go pour gérer les véhicules et les interventions d'un garage automobile.
L'API utilise `net/http` et une base SQLite avec `modernc.org/sqlite`, sans dépendance C.

Le projet se limite aux sept endpoints du sujet. Les tests et le calcul concurrent
des statistiques correspondent aux bonus proposés.

## Lancer le serveur

Prérequis : [Go 1.27.3 ou plus récent](https://go.dev/dl/).
Aucun serveur de base de données ni compilateur C n'est nécessaire.

Depuis la racine du dépôt :

```sh
go mod download
go run ./cmd/api
```

L'API écoute sur `http://localhost:8080`. Le fichier `garage.db` est créé au
premier démarrage et conserve les données entre les lancements. `Ctrl+C` arrête
le serveur.

Si le démarrage indique `address already in use`, une autre application utilise
déjà le port 8080. Il faut libérer ce port avant de lancer l'API, qui conserve
le port imposé par le sujet. Dans PowerShell, le processus peut être identifié avec :

```powershell
Get-NetTCPConnection -LocalPort 8080 -State Listen |
  Select-Object LocalAddress, LocalPort, OwningProcess
```

Pour choisir un autre fichier SQLite, définir `DB_PATH` avant le lancement.
Exemple dans PowerShell :

```powershell
$env:DB_PATH = "garage-test.db"
go run ./cmd/api
```

Pour produire un exécutable :

```sh
go build -o bin/garage.exe ./cmd/api
```

Sous Windows, il se lance avec `.\bin\garage.exe`. Sous Linux ou macOS,
utiliser `go build -o bin/garage ./cmd/api`, puis `./bin/garage`.

## Routes

| Méthode | Route | Réponse en cas de succès |
| --- | --- | --- |
| GET | `/vehicules` | 200, liste des véhicules |
| POST | `/vehicules` | 201, véhicule créé et en-tête `Location` |
| GET | `/vehicules/{plaque}` | 200, véhicule demandé |
| DELETE | `/vehicules/{plaque}` | 204, sans corps |
| POST | `/vehicules/{plaque}/interventions` | 201, intervention créée |
| GET | `/vehicules/{plaque}/interventions` | 200, liste des interventions |
| GET | `/stats` | 200, nombre de véhicules et interventions par statut |

Les listes vides sont renvoyées sous la forme `[]`. Une liste d'interventions
sur un véhicule inexistant renvoie 404.

### Ajouter un véhicule

Envoyer `POST /vehicules` avec `Content-Type: application/json` :

```json
{
  "plaque": "AA-123-BB",
  "marque": "Renault",
  "modele": "Clio",
  "annee": 2020,
  "kilometrage": 42000,
  "proprietaire": {
    "nom": "Camille Martin",
    "telephone": "0612345678"
  }
}
```

### Enregistrer une intervention

Envoyer `POST /vehicules/AA-123-BB/interventions` :

```json
{
  "date": "2024-03-01",
  "description": "Vidange et remplacement du filtre",
  "statut": "en_cours"
}
```

La plaque est prise dans l'URL et l'identifiant est généré par SQLite.
Il ne faut pas fournir ces deux champs dans le corps de création.

### Consulter les statistiques

Après ces deux créations, `GET /stats` renvoie :

```json
{
  "nombre_vehicules": 1,
  "interventions_par_statut": {
    "en_cours": 1,
    "terminee": 0,
    "annulee": 0
  }
}
```

Les sept requêtes sont disponibles dans [requests.http](requests.http).
Ce fichier peut être ouvert avec un client HTTP compatible, ou servir d'exemple
pour Postman. Dans PowerShell, une consultation simple se fait ainsi :

```powershell
Invoke-RestMethod http://localhost:8080/stats
```

## Validation et erreurs

- Plaque en majuscules au format `AA-123-AA`, unique dans la base.
- Marque, modèle, nom et téléphone du propriétaire non vides.
- Année comprise entre 1886 et l'année actuelle ; kilométrage positif ou nul.
- Date réelle au format `YYYY-MM-DD` et description non vide.
- Statut obligatoire : `en_cours`, `terminee` ou `annulee`.
- Suppression interdite dès qu'une intervention est liée au véhicule, même annulée.
- Corps JSON limité à 1 Mio ; JSON mal formé et champs inconnus refusés.

Les erreurs ont la forme `{"erreur":"message explicatif"}` :

| Code | Situation |
| --- | --- |
| 400 | JSON ou donnée invalide |
| 404 | Véhicule ou route introuvable |
| 405 | Méthode non autorisée, avec en-tête `Allow` |
| 409 | Plaque déjà enregistrée ou suppression interdite |
| 413 | Corps de requête trop volumineux |
| 415 | `Content-Type` différent de `application/json` |
| 500 | Erreur interne, détails conservés dans les logs |

## Organisation du code

```text
cmd/api/                Démarrage du serveur
internal/apperrors/     Sentinelles et erreur de validation personnalisée
internal/model/         Structures JSON et validation des données
internal/repository/    Requêtes SQL et création des tables SQLite
internal/service/       Opérations métier et statistiques concurrentes
internal/handler/       Routage net/http et lecture/écriture du JSON
requests.http           Exemples des sept endpoints
```

Un handler traduit une requête HTTP, appelle le service, puis produit une réponse
JSON. Le service valide les données et appelle le repository. SQLite impose aussi
l'unicité des plaques et les clés étrangères, pour que les contraintes restent
respectées lorsque plusieurs requêtes arrivent simultanément.

`Vehicule` intègre `Proprietaire` comme champ anonyme (embedding). Le tag
`json:"proprietaire"` conserve les coordonnées dans un objet JSON imbriqué.
La validation utilise des récepteurs valeur ; le service et le repository utilisent
des récepteurs pointeur avec des champs privés. Les erreurs sont propagées avec
`%w`, puis reconnues par `errors.Is` et `errors.As` pour choisir le code HTTP.

Les deux comptages de `/stats` sont lancés dans des goroutines et leurs résultats
passent par un channel. SQLite utilise une seule connexion pour sérialiser les
accès à la base. Les comptages restent deux lectures distinctes : les statistiques
ne constituent pas un instantané transactionnel si des écritures ont lieu en même temps.

## Vérifier le projet

```sh
go fmt ./...
go vet ./...
go test ./...
```

Chaque package contient un fichier de tests. Ils couvrent la validation avec des
tables et `t.Run`, les contraintes SQLite, la persistance après réouverture,
les statistiques concurrentes et les réponses HTTP avec `httptest`.
Les tests utilisent des bases temporaires et ne modifient pas `garage.db`.
