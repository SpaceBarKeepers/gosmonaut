package legacy

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/shaj13/go-guardian/v2/auth"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/aws/aws-sdk-go/aws"
)

func configureSpacesClient() (client *s3.S3) {
	key := goDotEnvVariable("SPACES_KEY")
	secret := goDotEnvVariable("SPACES_SECRET")

	s3Config := &aws.Config{
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:    aws.String("https://fra1.digitaloceanspaces.com"),
		Region:      aws.String("us-east-1"),
	}

	newSession, _ := session.NewSession(s3Config)
	s3Client := s3.New(newSession)

	return s3Client
}

func getAdmin(w http.ResponseWriter, r *http.Request) {
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

	cursor, err := client.Database(customerIdentifier).Collection("contentTypes").Find(ctx, bson.M{})
	var contentTypes []bson.M
	err = cursor.All(ctx, &contentTypes)
	if err != nil {
		log.Print(err)
	}
	err = json.NewEncoder(w).Encode(contentTypes)
	if err != nil {
		log.Print(err)
	}
}

func getUniversalCustomerId(userName string) string {
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
	var results *UniversalUser

	cursor := client.Database("universal").Collection("login")
	err = cursor.FindOne(ctx, bson.D{{"login", userName}}).Decode(&results)
	if err != nil {
		log.Print(err)
	}

	return results.Customer
}

func getUser(userName string, universalCustomerId string) (int, string, string, []byte) {
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
	var results *User

	cursor := client.Database(universalCustomerId).Collection("users")
	err = cursor.FindOne(ctx, bson.D{{"login", userName}}).Decode(&results)
	if err != nil {
		log.Print(err)
	}

	return results.Id, results.Login, results.Email, results.Password
}
