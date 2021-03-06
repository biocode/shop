package customer

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/foomo/shop/configuration"
	"github.com/foomo/shop/persistence"
	"github.com/foomo/shop/utils"
	"github.com/foomo/shop/version"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/mgo.v2/bson"
)

// !! NOTE: customer must not import order !!
//------------------------------------------------------------------
// ~ CONSTANTS / VARS
//------------------------------------------------------------------

const (
	VERBOSE = false
)

var globalCustomerPersistor *persistence.Persistor
var globalCustomerVersionsPersistor *persistence.Persistor
var globalCredentialsPersistor *persistence.Persistor

//------------------------------------------------------------------
// ~ PUBLIC METHODS
//------------------------------------------------------------------

// Returns GLOBAL_PERSISTOR. If GLOBAL_PERSISTOR is nil, a new persistor is created, set as GLOBAL_PERSISTOR and returned
func GetCustomerPersistor() *persistence.Persistor {
	url := configuration.MONGO_URL
	collection := configuration.MONGO_COLLECTION_CUSTOMERS
	if globalCustomerPersistor == nil {
		p, err := persistence.NewPersistor(url, collection)
		if err != nil || p == nil {
			panic(errors.New("failed to create mongoDB global persistor: " + err.Error()))
		}
		globalCustomerPersistor = p
		return globalCustomerPersistor
	}

	if url == globalCustomerPersistor.GetURL() && collection == globalCustomerPersistor.GetCollectionName() {
		return globalCustomerPersistor
	}

	p, err := persistence.NewPersistor(url, collection)
	if err != nil || p == nil {
		panic(err)
	}
	globalCustomerPersistor = p
	return globalCustomerPersistor
}

// Returns GLOBAL_PERSISTOR. If GLOBAL_PERSISTOR is nil, a new persistor is created, set as GLOBAL_PERSISTOR and returned
func GetCustomerVersionsPersistor() *persistence.Persistor {
	url := configuration.MONGO_URL
	collection := configuration.MONGO_COLLECTION_CUSTOMERS_HISTORY
	if globalCustomerVersionsPersistor == nil {
		p, err := persistence.NewPersistor(url, collection)
		if err != nil || p == nil {
			panic(errors.New("failed to create mongoDB order persistor: " + err.Error()))
		}
		globalCustomerVersionsPersistor = p
		return globalCustomerVersionsPersistor
	}

	if url == globalCustomerVersionsPersistor.GetURL() && collection == globalCustomerVersionsPersistor.GetCollectionName() {
		return globalCustomerVersionsPersistor
	}

	p, err := persistence.NewPersistor(url, collection)
	if err != nil || p == nil {
		panic(err)
	}
	globalCustomerVersionsPersistor = p
	return globalCustomerVersionsPersistor
}

// Returns GLOBAL_PERSISTOR. If GLOBAL_PERSISTOR is nil, a new persistor is created, set as GLOBAL_PERSISTOR and returned
func GetCredentialsPersistor() *persistence.Persistor {
	url := configuration.MONGO_URL
	collection := configuration.MONGO_COLLECTION_CREDENTIALS
	if globalCredentialsPersistor == nil {
		p, err := persistence.NewPersistor(url, collection)
		if err != nil || p == nil {
			panic(errors.New("failed to create mongoDB order persistor: " + err.Error()))
		}
		globalCredentialsPersistor = p
		return globalCredentialsPersistor
	}

	if url == globalCredentialsPersistor.GetURL() && collection == globalCredentialsPersistor.GetCollectionName() {
		return globalCredentialsPersistor
	}

	p, err := persistence.NewPersistor(url, collection)
	if err != nil || p == nil {
		panic(err)
	}
	globalCredentialsPersistor = p
	return globalCredentialsPersistor
}

// AlreadyExistsInDB checks if a customer with given customerID already exists in the database
func AlreadyExistsInDB(customerID string) (bool, error) {
	p := GetCustomerPersistor()
	q := p.GetCollection().Find(&bson.M{"id": customerID})
	count, err := q.Count()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func Count(query *bson.M, customProvider CustomerCustomProvider) (count int, err error) {
	return GetCustomerPersistor().GetCollection().Find(query).Count()
}

// Find returns an iterator for all entries found matching on query.
func Find(query *bson.M, customProvider CustomerCustomProvider) (iter func() (cust *Customer, err error), err error) {
	p := GetCustomerPersistor()
	_, err = p.GetCollection().Find(query).Count()
	if err != nil {
		log.Println(err)
	}
	q := p.GetCollection().Find(query)

	_, err = q.Count()
	if err != nil {
		return
	}
	mgoiter := q.Iter()
	iter = func() (cust *Customer, err error) {
		cust = &Customer{}
		if mgoiter.Next(cust) {
			return mapDecode(cust, customProvider)
		}
		return nil, nil
	}
	return
}

func UpsertCustomer(c *Customer) error {

	// order is unlinked or not yet inserted in db
	if c.unlinkDB || c.BsonId == "" {
		return nil
	}
	p := GetCustomerPersistor()

	// Get current version from db and check against verssion of c
	// If they are not identical, there must have been another upsert which would be overwritten by this one.
	// In this case upsert is skipped and an error is returned,
	customerLatestFromDb := &Customer{}
	err := p.GetCollection().Find(&bson.M{"_id": c.BsonId}).Select(&bson.M{"version": 1}).One(customerLatestFromDb)

	if err != nil {
		log.Println("Upsert failed: Could not find customer with id", c.GetID(), "Error:", err)

		return err
	}

	latestVersionInDb := customerLatestFromDb.Version.GetVersion()
	if latestVersionInDb != c.Version.GetVersion() && !c.Flags.forceUpsert {
		errMsg := fmt.Sprintln("WARNING: Cannot upsert latest version ", strconv.Itoa(latestVersionInDb), "in db with version", strconv.Itoa(c.Version.GetVersion()), "!")
		log.Println(errMsg)
		return errors.New(errMsg)
	}

	if c.Flags.forceUpsert {
		// Remember this number, so that we later know from which version we came from
		v := c.Version.Current
		// Set the current version number to keep history consistent
		c.Version.Current = latestVersionInDb
		c.Version.Increment()
		c.Flags.forceUpsert = false
		// Overwrite NumberPrevious, to remember where we came from
		c.Version.Previous = v
	} else {
		c.Version.Increment()
	}

	_, err = p.GetCollection().UpsertId(c.BsonId, c)
	if err != nil {
		return err
	}

	// Store version in history
	bsonId := c.BsonId
	c.BsonId = "" // Temporarily reset Mongo ObjectId, so that we can perfrom an Insert.
	pHistory := GetCustomerVersionsPersistor()
	pHistory.GetCollection().Insert(c)
	c.BsonId = bsonId // restore bsonId

	return err
}

func UpsertAndGetCustomer(c *Customer, customProvider CustomerCustomProvider) (*Customer, error) {
	err := UpsertCustomer(c)
	if err != nil {
		return nil, err
	}
	return GetCustomerById(c.GetID(), customProvider)
}

func DeleteCustomer(c *Customer) error {
	err := GetCustomerPersistor().GetCollection().Remove(bson.M{"_id": c.BsonId})
	if err != nil {
		return err
	}
	err = DeleteCredential(c.Email)

	return err
}
func DeleteCustomerById(id string) error {
	customer, err := GetCustomerById(id, nil)
	if err != nil {
		return err
	}
	DeleteCustomer(customer)
	return err
}

func GetCustomerByQuery(query *bson.M, customProvider CustomerCustomProvider) (*Customer, error) {
	return findOneCustomer(query, nil, "", customProvider, false)
}

// GetCustomerById returns the customer with id
func GetCustomerById(id string, customProvider CustomerCustomProvider) (*Customer, error) {
	return findOneCustomer(&bson.M{"id": id}, nil, "", customProvider, false)
}

// GetCustomerByEmail // TODO this won't work for guests, because for guests there could be multiple entries for the same email address
func GetCustomerByEmail(email string, customProvider CustomerCustomProvider) (*Customer, error) {
	return findOneCustomer(&bson.M{"email": email}, nil, "", customProvider, false)
}
func GetCurrentCustomerByIdFromVersionsHistory(customerId string, customProvider CustomerCustomProvider) (*Customer, error) {
	return findOneCustomer(&bson.M{"id": customerId}, nil, "-version.current", customProvider, true)
}
func GetCurrentVersionOfCustomerFromVersionsHistory(customerId string) (*version.Version, error) {
	customer, err := findOneCustomer(&bson.M{"id": customerId}, &bson.M{"version": 1}, "-version.current", nil, true)
	if err != nil {
		return nil, err
	}
	return customer.GetVersion(), nil
}
func GetCustomerByVersion(customerId string, version int, customProvider CustomerCustomProvider) (*Customer, error) {
	return findOneCustomer(&bson.M{"id": customerId, "version.current": version}, nil, "", customProvider, true)
}

func Rollback(customerId string, version int) error {
	currentCustomer, err := GetCustomerById(customerId, nil)
	if err != nil {
		return err
	}
	if version >= currentCustomer.GetVersion().Current || version < 0 {
		return errors.New("Cannot perform rollback to " + strconv.Itoa(version) + " from version " + strconv.Itoa(currentCustomer.GetVersion().Current))
	}
	customerFromVersionsHistory, err := GetCustomerByVersion(customerId, version, nil)
	if err != nil {
		return err
	}
	// Set bsonId from current customer to customer from history to overwrite current customer on next upsert.
	customerFromVersionsHistory.BsonId = currentCustomer.BsonId
	customerFromVersionsHistory.Flags.forceUpsert = true
	return customerFromVersionsHistory.Upsert()

}

func DropAllCustomers() error {
	return GetCustomerPersistor().GetCollection().DropCollection()

}
func DropAllCredentials() error {
	return GetCredentialsPersistor().GetCollection().DropCollection()
}

func DropAllCustomersAndCredentials() error {
	err := GetCustomerPersistor().GetCollection().DropCollection()
	if err != nil {
		return err
	}
	return GetCredentialsPersistor().GetCollection().DropCollection()

}

//------------------------------------------------------------------
// ~ PRIVATE METHODS
//------------------------------------------------------------------

// findOneCustomer returns one Customer from the customer database or from the customer history database
func findOneCustomer(find *bson.M, selection *bson.M, sort string, customProvider CustomerCustomProvider, fromHistory bool) (*Customer, error) {
	var p *persistence.Persistor
	if fromHistory {
		p = GetCustomerVersionsPersistor()
	} else {
		p = GetCustomerPersistor()
	}
	customer := &Customer{}
	if find == nil {
		find = &bson.M{}
	}
	if selection == nil {
		selection = &bson.M{}
	}
	if sort != "" {
		err := p.GetCollection().Find(find).Select(selection).Sort(sort).One(customer)
		if err != nil {
			return nil, err
		}
	} else {
		err := p.GetCollection().Find(find).Select(selection).One(customer)
		if err != nil {
			return nil, err
		}
	}
	if customProvider != nil {
		var err error
		customer, err = mapDecode(customer, customProvider)
		if err != nil {
			return nil, err
		}
	}
	if customer == nil {
		return nil, errors.New("No result for " + utils.ToJSON(find))
	}

	return customer, nil
}

// insertCustomer inserts a customer into the database
func insertCustomer(c *Customer) error {
	p := GetCustomerPersistor()
	alreadyExists, err := AlreadyExistsInDB(c.GetID())
	if err != nil {
		return err
	}
	if alreadyExists {
		log.Println("User with id", c.GetID(), "already exists in the database!")
		return nil
	}
	err = p.GetCollection().Insert(c)
	if err != nil {
		return err
	}
	pHistory := GetCustomerVersionsPersistor()
	err = pHistory.GetCollection().Insert(c)

	return err
}

// mapDecode maps interfaces to specific types provided by customProvider
func mapDecode(cust *Customer, customProvider CustomerCustomProvider) (customer *Customer, err error) {
	/* Map CustomerCustom */
	customerCustom := customProvider.NewCustomerCustom()
	if customerCustom != nil && cust.Custom != nil {
		err = mapstructure.Decode(cust.Custom, customerCustom)
		if err != nil {
			return nil, err
		}
		cust.Custom = customerCustom
	}

	/* Map AddressCustom */
	for _, address := range cust.Addresses {
		addressCustom := customProvider.NewAddressCustom()
		if addressCustom != nil && address.Custom != nil {

			err = mapstructure.Decode(address.Custom, addressCustom)
			if err != nil {
				return nil, err
			}
			address.Custom = addressCustom
		}
	}
	return cust, nil
}
