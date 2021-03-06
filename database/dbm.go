package dbm

import (
	"errors"
	"gotools/reflectutil"
	"log"

	"math/rand"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	ErrNotFound = errors.New("not found")
	ErrCursor   = errors.New("invalid cursor")
)

type DBError struct {
	Errs []error
}

func (err *DBError) Error() string {
	var desc = ""
	for _, er := range err.Errs {
		desc += " " + er.Error()
	}
	return desc
}

type DatabaseQuerier interface {
	GetModels(mongoQuery M, resultInterface interface{}, limit int, skip int) error
	GetOneModel(mongoQuery M, result interface{}) error
	AggregateModels(mongoQuery []M, resultInterface interface{}, collectionName string) error
	InsertModel(model ...interface{}) error
	UpdateModelId(Id interface{}, model interface{}) error
	Update(query M, update M, model interface{}) (interface{}, error)
	RemoveModel(mongoQuery M, model interface{}) error
	//TODO: Create RemoveModel
	//TODO: make isExist work
	// IsExist(result interface{}) bool
}

type M map[string]interface{}

type MongoDatabaseSession struct {
	// host     string
	// port     string
	// db_name string
	// username string
	// password string

	url      string
	Database *mgo.Database
}

type Modeler interface {
	SetName(name string)
	Name() string
}

func NewMongoDatabaseSessionInfo(host, port, db_name, username, password string) *MongoDatabaseSession {
	url := "mongodb://" + username + ":" + password + "@" + host + ":" + port + "/" + db_name
	return &MongoDatabaseSession{url: url}
}

func NewMongoDatabaseSession(url string) *MongoDatabaseSession {
	return &MongoDatabaseSession{url: url}
}

func configureMongoDatabaseSession(session *mgo.Session) {

}

// func (db *MongoDatabaseSession) getOrCreateCollection(collectionName string) *db.Collection {
// 	defer func (collectionName) *mgo.Collection {
// 		if r := recover(); r != nil {
// 			db.Database.
// 		}
// 	}
// }

func (db *MongoDatabaseSession) Connect() error {
	// log.Println("DB URL = " + db.host + ":" + db.port)
	//"mongodb://" + db.username + ":" + db.password +"@" +
	log.Println("Mongo trying to connect to", db.url)
	session, err := mgo.Dial(db.url)
	// session, err := mgo.DialWithInfo(&mgo.DialInfo{Addrs: []string{db.host + ":" + db.port}, Username: db.username, Password: db.password, Database: db.db_name})
	if err != nil {
		panic(err)
	}
	configureMongoDatabaseSession(session)
	db.Database = session.DB("")
	return err
}

func (db *MongoDatabaseSession) Close() {
	db.Database.Session.Close()
}

//GetRandomOneModel get one model random in all documents
func (db *MongoDatabaseSession) GetRandomOneModel(model interface{}) error {
	collectionName := reflectutil.GetInnerTypeName(model)
	collection := db.Database.C(collectionName)
	countElem, err := collection.Count()
	if err != nil {
		return err
	}
	elemNum := rand.Intn(countElem)
	collection.Find(bson.M{}).Skip(elemNum).One(model)
	return nil
}

//GetModels retrieves all the data from mongoDB
//mongoQuery is the query from MongoDB query
//resultInterface is a slice representing the model required, it will be fill with the result of the query
//limit of result if limit < 0 no limit used
//skip corresponding the number elements to skip
//return an err if soimething bad appened
func (db *MongoDatabaseSession) GetModels(mongoQuery M, result interface{}, limit int, skip int) error {

	collectionName := reflectutil.GetInnerTypeName(result)
	collection := db.Database.C(collectionName)
	// result := reflectutil.CreatePtrToSliceFromInterface(resultInterface)
	var err error = nil
	switch {
	case limit <= 0 && skip <= 0:
		err = collection.Find(bson.M(mongoQuery)).All(result)
	case limit > 0 && skip <= 0:
		err = collection.Find(bson.M(mongoQuery)).Limit(limit).All(result)
	case limit <= 0 && skip > 0:
		err = collection.Find(bson.M(mongoQuery)).Skip(skip).All(result)
	case limit > 0 && skip > 0:
		err = collection.Find(bson.M(mongoQuery)).Skip(skip).Limit(limit).All(result)
	}
	return err
}

func (db *MongoDatabaseSession) AggregateModels(mongoQuery []M, resultInterface interface{}, collectionName string) error {
	if collectionName == "" {
		collectionName = reflectutil.GetInnerTypeName(resultInterface)
	}
	collection := db.Database.C(collectionName)
	// groupByQ := make([]bson.M, 0, len(groupBy))
	// for _, m := range groupBy {
	// 	groupByQ = append(groupByQ, bson.M{"$group": m})
	// }
	err := collection.Pipe(mongoQuery).All(resultInterface)
	return err
}

func (db *MongoDatabaseSession) GetOneModel(mongoQuery M, result interface{}) error {
	collectionName := reflectutil.GetInnerTypeName(result)
	collection := db.Database.C(collectionName)
	err := collection.Find(bson.M(mongoQuery)).One(result)
	return err
}

// func (db *MongoDatabaseSession) GetModel(collectionName string, mongoQuery M, result interface{}) error {
// 	collection := db.Database.C(collectionName)
// 	err := collection.Find(bson.M(mongoQuery)).All(result)
// 	return err
// }

func (db *MongoDatabaseSession) InsertModel(model ...interface{}) error {
	sortedModel := reflectutil.SortArrayByType(model)
	err := &DBError{}
	err.Errs = make([]error, 0)
	for collectionName, models := range sortedModel {
		collection := db.Database.C(collectionName)
		errTmp := collection.Insert(models...)
		if errTmp != nil {
			err.Errs = append(err.Errs, errTmp)
		}
	}
	if len(err.Errs) > 0 {
		return err
	}
	return nil
}

func (db *MongoDatabaseSession) Update(query M, update M, model interface{}) (interface{}, error) {
	collectionName := reflectutil.GetInnerTypeName(model)
	collection := db.Database.C(collectionName)
	change := mgo.Change{
		Update:    update,
		Upsert:    false,
		ReturnNew: true,
	}
	res, err := collection.Find(query).Apply(change, model)
	return res, err
}

func (db *MongoDatabaseSession) UpdateModelId(id interface{}, model interface{}) error {
	collectionName := reflectutil.GetInnerTypeName(model)
	collection := db.Database.C(collectionName)
	_, err := collection.UpsertId(id, model)
	return err
}

func (db *MongoDatabaseSession) RemoveModel(mongoQuery M, model interface{}) error {
	collectionName := reflectutil.GetInnerTypeName(model)
	collection := db.Database.C(collectionName)
	_, err := collection.RemoveAll(mongoQuery)
	return err
}
