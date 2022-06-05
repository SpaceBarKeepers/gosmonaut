package legacy

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/shaj13/go-guardian/v2/auth"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"time"
)

func getContentLib(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var customerIdentifier string
	var public bool
	if len(vars["customerIdentifier"]) > 0 {
		customerIdentifier = vars["customerIdentifier"]
		public = true
	} else {
		customerIdentifier = auth.UserFromCtx(r.Context()).GetGroups()[0]
		public = false
	}

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

	var results []bson.M
	if public == true {
		cursor, err := client.Database(customerIdentifier).Collection(vars["contentType"]).Find(ctx, bson.D{{"published", true}})
		err = cursor.All(ctx, &results)
		if err != nil {
			log.Print(err)
		}
	} else {
		cursor, err := client.Database(customerIdentifier).Collection(vars["contentType"]).Find(ctx, bson.M{})
		err = cursor.All(ctx, &results)
		if err != nil {
			log.Print(err)
		}
	}
	log.Println(results)
	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		log.Print(err)
	}
}

func postContentItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	decoder := json.NewDecoder(r.Body)
	var formData map[string]interface{}
	err := decoder.Decode(&formData)
	if err != nil {
		panic(err)
	}
	log.Println(formData)

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

	cursorContentTypeDef := client.Database(customerIdentifier).Collection("contentTypes")
	var typesResult map[string]interface{}

	err = cursorContentTypeDef.FindOne(ctx, bson.D{{"name", vars["contentType"]}}).Decode(&typesResult)
	if err != nil {
		log.Print(err)
	}

	log.Println(typesResult["fields"])

	insertData := make(map[string]interface{})

	for _, fieldMapValue := range typesResult["fields"].(primitive.A) {
		fieldId, fieldType := fieldMapValue.(map[string]interface{})["id"], fieldMapValue.(map[string]interface{})["type"]
		switch fieldType {
		case "textfield":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "textarea":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "datetime":
			if formData[fieldId.(string)] != nil {
				log.Println(formData[fieldId.(string)].(string))
				timeToInsert, errorParsingTime := time.Parse("2006-01-02T15:04:05-07:00", formData[fieldId.(string)].(string))
				if errorParsingTime != nil {
					timeToInsert, errorParsingTime = time.Parse("2006-01-02T15:04:05.000Z", formData[fieldId.(string)].(string))
					if errorParsingTime != nil {
						timeToInsert, errorParsingTime = time.Parse("2006-01-02T15:04:05Z", formData[fieldId.(string)].(string))
						log.Println("Failed to parse even tertiary time format.")
					}
				}
				insertData[fieldId.(string)] = primitive.NewDateTimeFromTime(timeToInsert)
			} else {
				insertData[fieldId.(string)] = nil
			}
		case "image":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "media":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "dropdown":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "gallery":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "link":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "checkboxWithTwoTextFields":
			if formData[fieldId.(string)] != nil {
				insertData[fieldId.(string)] = formData[fieldId.(string)]
			} else {
				// in case no data are provided, create [false, null, null]
				var defaultCheckboxWithTwoTextFields [3]interface{}
				defaultCheckboxWithTwoTextFields[0] = false
				insertData[fieldId.(string)] = defaultCheckboxWithTwoTextFields
			}

		case "toggle":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "richtextEditor":
			insertData[fieldId.(string)] = formData[fieldId.(string)]
		case "datePublished":
			if formData["datePublished"] != nil {
				insertData["datePublished"] = formData["datePublished"]
			} else {
				insertData["datePublished"] = time.Now()
			}
		default:
			log.Printf("%s is not defined as field", fieldId)
		}
	}

	if formData["id"] != nil {
		insertData["id"] = formData["id"]
		cursorContentTypeColl := client.Database(customerIdentifier).Collection(vars["contentType"])

		dataForDb, err := bson.Marshal(insertData)
		if err != nil {
			log.Print(err)
		}

		postResult, err := cursorContentTypeColl.ReplaceOne(
			ctx,
			bson.M{"id": formData["id"]},
			dataForDb,
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(postResult)
		err = json.NewEncoder(w).Encode(responseOK)
		if err != nil {
			log.Print(err)
		}
	} else {
		cursorContentTypeColl := client.Database(customerIdentifier).Collection(vars["contentType"])

		var lastDocument = make(map[string]interface{})
		opts := options.FindOne().SetSort(bson.D{{"id", -1}})
		err = cursorContentTypeColl.FindOne(ctx, bson.D{}, opts).Decode(&lastDocument)
		if err != nil {
			log.Print(err)
			lastDocument["id"] = int32(0)
		}

		assertedValue, ok := lastDocument["id"].(int32)
		if !ok {
			assertedValue = int32(lastDocument["id"].(float64))
		}
		insertData["id"] = assertedValue + 1

		dataForDb, err := bson.Marshal(insertData)
		if err != nil {
			log.Print(err)
		}

		postResult, err := cursorContentTypeColl.InsertOne(ctx, dataForDb)
		if err != nil {
			log.Print(err)
		}
		log.Println(postResult)
		err = json.NewEncoder(w).Encode(responseOK)
		if err != nil {
			log.Print(err)
		}

		log.Println(insertData)
	}

}

func deleteContentItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	customerIdentifier := auth.UserFromCtx(r.Context()).GetGroups()[0]

	// In case request have no body, return 400
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	// Decode body to JSON
	decoder := json.NewDecoder(r.Body)
	var req RequestContentId
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
	cursor := client.Database(customerIdentifier).Collection(vars["contentType"])
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
}
