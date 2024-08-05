# AiGoGo 
AiGoGo is your intelligent companion that safeguards cherished memories, suggests activities tailored to your unique preferences, and incorporates local knowledge for a truly personalized experience.

## Overview
- Create embeddings file
- Run the inference website

## Creating the embeddings file
The output file is located in cmd/aigogo/vecdb/embeddings.gob .

The source csv files is pointed to by the RAGCSV environment variable.

Eg. 
```
export RAGCSV="/home/siuyin/Downloads/aigogo data - General.csv"
```

Create it with:
```
cd cmd/loadRAGCSV
go run main.go
```

## Developement run
```
cd cmd/aigogo
go run main.go
```

## Production build, deploy and run
```
export DEPLOY=PROD
export API_KEY=....
export MAPS_API_KEY=....

skaffold build   # this creates an image ready to deploy
```
