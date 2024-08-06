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
mkdir -p /data/aigogo/123456
cp data/aigogo/123456/* /data/aigogo/123456
cd cmd/aigogo
go run main.go
```

## Testing
Set the TESTING environement variable to skip calling Gen AI services.

Note: `func init()` is still called and the application initialized.
```
cd cmd/aigogo
TESTING=1 go test
```

## Production build, deploy and run
```
export DEPLOY=PROD
export API_KEY=....
export MAPS_API_KEY=....

skaffold build   # this creates an image ready to deploy
```

Create a Cloud Storage bucket and have it mounted on /data
copy data/aigogo/123456 to /data/aigogo/123456
