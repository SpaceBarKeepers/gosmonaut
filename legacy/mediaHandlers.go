package legacy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/shaj13/go-guardian/v2/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func getMedia(w http.ResponseWriter, r *http.Request) {
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	uri := goDotEnvVariable("MAIN_DB")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	cursor, err := client.Database(customerIdentifier).Collection("media").Find(ctx, bson.M{})
	var results []bson.M
	err = cursor.All(ctx, &results)
	if err != nil {
		log.Fatal(err)
	}
	_ = json.NewEncoder(w).Encode(results)
}

func getMediaItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var customerIdentifier string
	if len(vars["customerIdentifier"]) > 0 {
		customerIdentifier = vars["customerIdentifier"]
	} else {
		customerIdentifier = auth.UserFromCtx(r.Context()).GetGroups()[0]
	}

	client := configureSpacesClient()

	input := &s3.GetObjectInput{
		Bucket: aws.String(goDotEnvVariable("SPACES_PREFIX") + customerIdentifier),
		Key:    aws.String(vars["mediaIdentifier"]),
	}

	result, err := client.GetObject(input)
	if err != nil {
		fmt.Println(err.Error())
	}

	mediaFile, _ := ioutil.ReadAll(result.Body)
	_, _ = w.Write(mediaFile)
}

func postMedia(w http.ResponseWriter, r *http.Request) {
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	// 10 << 20 specifies a maximum upload of 10 MB files.
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// FormFile returns the first file for the given key `uploadFile`
	file, handler, err := r.FormFile("uploadFile")
	if err != nil {
		log.Println("Error Retrieving the File")
		log.Println(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	log.Printf("Uploaded File: %+v\n", handler.Filename)
	log.Printf("File Size: %+v\n", handler.Size)
	log.Printf("MIME Header: %+v\n", handler.Header)

	// upload to DO space
	client := configureSpacesClient()
	fileNamePrepared := strings.Join(strings.Fields(handler.Filename[:len(handler.Filename)-len(filepath.Ext(handler.Filename))]), "")
	fileNameForSpace := fileNamePrepared + strconv.Itoa(int(time.Now().Unix()))
	fileExtForSpace := strings.ToLower(filepath.Ext(handler.Filename))

	object := s3.PutObjectInput{
		Bucket: aws.String(goDotEnvVariable("SPACES_PREFIX") + customerIdentifier),
		Key:    aws.String(fmt.Sprintf("%s%s", fileNameForSpace, fileExtForSpace)),
		Body:   file,
		ACL:    aws.String("private"),
		Metadata: map[string]*string{
			"x-amz-meta-my-key": aws.String("be-upload"),
		},
	}
	_, err = client.PutObject(&object)
	if err != nil {
		log.Println(err.Error())
		return
	}

	//add Media Item
	uri := goDotEnvVariable("MAIN_DB")
	clientMongo, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = clientMongo.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	err = clientMongo.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	cursorMedia := clientMongo.Database(customerIdentifier).Collection("media")

	var lastDocument = make(map[string]interface{})
	opts := options.FindOne().SetSort(bson.D{{"id", -1}})
	err = cursorMedia.FindOne(ctx, bson.D{}, opts).Decode(&lastDocument)
	if err != nil {
		log.Print(err)
		lastDocument["id"] = int32(0)
	}

	var insertData MediaItem
	insertData.Id = lastDocument["id"].(int32) + 1
	insertData.Name = fileNameForSpace + fileExtForSpace
	insertData.Src = goDotEnvVariable("ROOT_DOMAIN") + "/" + customerIdentifier + "/v1/public/media/" + fileNameForSpace + fileExtForSpace
	insertData.Desc = ""
	insertData.Type = getContentTypeOfFile(fileExtForSpace)

	dataForDb, err := bson.Marshal(insertData)
	if err != nil {
		log.Print(err)
	}

	postResult, err := cursorMedia.InsertOne(ctx, dataForDb)
	if err != nil {
		log.Print(err)
	}

	log.Println(postResult)

	var response = ResponsePostMediaOK{ResponseCode: 1, Id: insertData.Id, Name: insertData.Name, Src: insertData.Src, Desc: insertData.Desc, Type: insertData.Type}
	payload, err := json.Marshal(response)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}

}

func deleteMedia(w http.ResponseWriter, r *http.Request) {
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	// In case request have no body, return 400
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	// Decode body to JSON
	decoder := json.NewDecoder(r.Body)
	var req RequestMediaIdAndFileName
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// start DB operation
	uri := goDotEnvVariable("MAIN_DB")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}
	// setup cursor
	cursor := client.Database(customerIdentifier).Collection("media")
	// fetch item object
	var itemObject map[string]interface{}
	err = cursor.FindOne(ctx, bson.M{"id": req.ItemId}).Decode(&itemObject)
	if err != nil {
		log.Print(err)
	}
	// delete item
	_, err = cursor.DeleteOne(ctx, bson.M{"id": req.ItemId})
	// if failed, return response900
	if err != nil {
		log.Print(err)

		payload, err := json.Marshal(response900)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(payload)
		if err != nil {
			log.Println(err)
		}
		return
	}
	// if success, return responseOK
	payload, err := json.Marshal(responseOK)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}

	clientSpace := configureSpacesClient()
	object := &s3.DeleteObjectInput{
		Bucket: aws.String(goDotEnvVariable("SPACES_PREFIX") + customerIdentifier),
		Key:    aws.String(itemObject["name"].(string)),
	}

	_, err = clientSpace.DeleteObject(object)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func patchMedia(w http.ResponseWriter, r *http.Request) {
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	decoder := json.NewDecoder(r.Body)
	var requestData MediaItem
	err := decoder.Decode(&requestData)
	if err != nil {
		panic(err)
	}
	log.Println(requestData)

	// start DB operation
	uri := goDotEnvVariable("MAIN_DB")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}
	// setup cursor
	cursor := client.Database(customerIdentifier).Collection("media")
	// delete item
	_, err = cursor.UpdateOne(ctx, bson.M{"id": requestData.Id}, bson.D{
		{"$set", bson.D{{"desc", requestData.Desc}}},
		{"$set", bson.D{{"name", requestData.Name}}},
		{"$set", bson.D{{"src", requestData.Src}}},
	})
	// if failed, return response900
	if err != nil {
		log.Print(err)

		payload, err := json.Marshal(response900)
		if err != nil {
			log.Println(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(payload)
		if err != nil {
			log.Println(err)
		}
		return
	}
	// if success, return responseOK
	payload, err := json.Marshal(responseOK)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Println(err)
	}

}
