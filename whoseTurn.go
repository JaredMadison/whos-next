package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	alexa "github.com/mikeflynn/go-alexa/skillserver"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var Applications = map[string]interface{}{
	"/echo/whoseturn": alexa.EchoApplication{ // Route
		AppID:   "amzn1.ask.skill.cf8d8857-9887-45e4-a71f-a734512c46e5",
		Handler: EchoWhoseTurn,
	},
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.Println("Starting program")
	//start application on port 7152
	alexa.Run(Applications, "7152")
}

//main handler for requests
func EchoWhoseTurn(w http.ResponseWriter, r *http.Request) {
	//echoReq := context.Get(r, "echoRequest").(*alexa.EchoRequest)
	echoReq := alexa.GetEchoRequest(r)
	log.Println(echoReq.GetRequestType())

	// Start Mongo
	mongodb, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	//access the correct user from the DB
	col := mongodb.DB("whoseTurn").C("users")
	defer mongodb.Close()

	//get user's id and load into user var
	id := echoReq.GetUserID()
	user := loadUser(col, id)

	//prints request type for debugging
	log.Println(echoReq.GetRequestType())
	if echoReq.GetRequestType() == "LaunchRequest" {
		msg := "Welcome to Who's Next. What can I do for you?"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)

		json, _ := echoResp.String()
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Write(json)

	} else if echoReq.GetRequestType() == "SessionEndedRequest" {
		msg := "Goodbye"
		//end the session in the case of a SessionWndedRequest
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(true)
		json, _ := echoResp.String()
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Write(json)

	} else if echoReq.GetRequestType() == "IntentRequest" {
		//print intent name for debugging
		log.Println(echoReq.GetIntentName())
		//create an echoResp that will be populated by intent function
		var echoResp *alexa.EchoResponse

		//call intent function depending on given intent name
		switch echoReq.GetIntentName() {
		case "AMAZON.HelpIntent":
			echoResp = help(echoReq)
		case "AMAZON.StopIntent":
			echoResp = cancel(echoReq)
		case "AMAZON.CancelIntent":
			echoResp = cancel(echoReq)
		case "ListActivities":
			echoResp = listActivities(echoReq, user)
		case "ListPeopleOnActivity":
			echoResp = listPeopleOnActivity(echoReq, col, user)
		case "AddActivity":
			echoResp = addActivity(echoReq, col, user)
		case "AddPersonToActivity":
			echoResp = addPersonToActivity(echoReq, col, user)
		case "RemoveActivity":
			echoResp = removeActivity(echoReq, col, user)
		case "RemovePersonFromActivity":
			echoResp = removePersonFromActivity(echoReq, col, user)
		case "WhoseTurnForActivity":
			echoResp = whoseTurnForActivity(echoReq, col, user)
		case "CompletedActivity":
			echoResp = completedActivity(echoReq, col, user)
		}
		json, _ := echoResp.String()
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.Write(json)
	}
}

//look for user and either return existing user or create new user and load in DB
func loadUser(col *mgo.Collection, userID string) *User {
	var user User
	err := col.Find(bson.M{"id": userID}).One(&user)

	if err != nil && err != mgo.ErrNotFound {
		log.Println("Failed: *************************** load user", err)
		return nil
	}
	if user.ID == "" {
		user.ID = userID
		fmt.Println("Loaded user")
		col.Insert(user)
	}
	//spew.Dump(user)
	return &user
}

//handles basic amazon built in cancel intent
//ends the session
func cancel(echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	msg := "Goodbye"
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(true)
	return echoResp
}

//handles basic amazon built in help intent
//responds with a helpful message
func help(echoReq *alexa.EchoRequest) *alexa.EchoResponse {
	msg := "Try adding an activity by saying add, then the name of the activity"
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//updates DB of a specific user
//should be called any time the DB is changed
func updateUser(col *mgo.Collection, user *User) {
	col.Upsert(bson.M{"id": user.ID}, user)
}

//function to list all activities a user has
func listActivities(echoReq *alexa.EchoRequest, user *User) *alexa.EchoResponse {
	msg := ""

	//switch depending on number of activities user has
	switch len(user.Activities) {

	case 0:
		msg = "You have no activities currently"

	case 1:
		msg = "You have one activity " + user.Activities[0].Name

	default:
		activitiesLen := strconv.Itoa(len(user.Activities))
		msg = "You have " + activitiesLen + " activities "

		//go through all activites and formulate a response message
		for index, activity := range user.Activities {
			if index == len(user.Activities)-1 {
				msg = msg + " and " + user.Activities[index].Name + " "
			} else {
				msg = msg + activity.Name + " "
			}
		}
	}

	//return a response with message of listed activities
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//lists all people currently added to a specific activity
func listPeopleOnActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	//get activity name from request
	activityName, errActivity := echoReq.GetSlotValue("activity")

	//if there was an error getting the activity slot value
	if errActivity != nil {
		log.Println("error")
		msg = "There was an error with your activity name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	//get index of specific activity
	activityIndex := getActivityIndex(user.Activities, activityName)

	switch len(user.Activities[activityIndex].People) {

	case 0:
		msg = "There is no one currently assigned to " + activityName

	case 1:
		msg = user.Activities[activityIndex].People[0] + " is the only one assigned to " + activityName

	default:
		peopleLen := strconv.Itoa(len(user.Activities[activityIndex].People))
		msg = "You have " + peopleLen + " people on this activity "

		//loop through all people on an activity and formulate a response message
		for index, person := range user.Activities[activityIndex].People {
			if index == len(user.Activities[activityIndex].People)-1 {
				msg = msg + " and " + person + " "
			} else {
				msg = msg + person + " "
			}
		}
	}

	//return a response with message of people on a specfiic activity
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp

}

//function that handles the completedActivity intent
//responds with error messages if the activity or person's name is not found in database
//otherwise, updates database accordingly and responds with a success response
func completedActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	//get activity name from request
	activityName, errActivity := echoReq.GetSlotValue("activity")
	personName, errPerson := echoReq.GetSlotValue("person")

	fmt.Println(activityName)
	if (errActivity != nil) || (activityName == "") {
		log.Println("error")
		msg = "There was an error finding that activity name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	if (errPerson != nil) || (personName == "") {
		log.Println("error")
		msg = "There was an error finding that persons name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	activityIndex := getActivityIndex(user.Activities, activityName)

	currentPersonIndex := user.Activities[activityIndex].WhoseCurrent
	//the person's name whose turn it is
	personTurn := user.Activities[activityIndex].People[currentPersonIndex]

	//check to see if it is the person's turn
	if personTurn == personName {
		if user.Activities[activityIndex].WhoseCurrent == len(user.Activities[activityIndex].People)-1 {
			user.Activities[activityIndex].WhoseCurrent = 0
		} else {
			user.Activities[activityIndex].WhoseCurrent++
		}
	} else {
		msg = "It is not " + personName + "'s turn to " + activityName
	}

	msg = personName + " has completed the activity, it is now " + user.Activities[activityIndex].People[user.Activities[activityIndex].WhoseCurrent] + "'s turn to " + activityName

	//update database
	updateUser(col, user)
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//handles the intent that tells the user whose turn it is for an activity
//responds with error messages if the activity is not found or the person is not assigned to it
func whoseTurnForActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	activityName, err := echoReq.GetSlotValue("activity")

	if err != nil {
		log.Println("error")
		msg = "There was an error adding them to your activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	activityIndex := getActivityIndex(user.Activities, activityName)

	if len(user.Activities[activityIndex].People) == 0 {
		msg = "There is no one currently assigned to this activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	currentPersonIndex := user.Activities[activityIndex].WhoseCurrent
	personTurn := user.Activities[activityIndex].People[currentPersonIndex]

	msg = "It is " + personTurn + "'s turn to " + activityName

	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//handles the intent to add someone to an activity
//responds with and error if there is no activity by that name or if they are already added
func addPersonToActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	actvityName, err := echoReq.GetSlotValue("Activity")
	personName, err := echoReq.GetSlotValue("person")

	if err != nil {
		log.Println("error")
		msg = "There was an error adding them to your activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	//activityIndex is used to tell if there is an activity by that name
	activityIndex := -1
	for index, activity := range user.Activities {
		if activity.Name == actvityName {
			activityIndex = index
		}
	}

	if activityIndex == -1 {
		log.Println("error: activity not found")
		msg = "Sorry there is no activity by that name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	//search activities to see if they are already added
	for _, name := range user.Activities[activityIndex].People {
		if personName == name {
			msg = personName + " is already added to " + actvityName
			echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
			return echoResp
		}
	}

	user.Activities[activityIndex].People = append(user.Activities[activityIndex].People, personName)
	updateUser(col, user)

	msg = "Added " + personName + " to " + actvityName
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//handles addActivity intent
//responds with error messages if the activity is already added or another error
func addActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	fmt.Println("In addActivity")
	activityName, err := echoReq.GetSlotValue("Activity")
	if err != nil {
		log.Println("error")
		msg = "There was an error adding your activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	fmt.Println("Activity name is " + activityName)

	if user.Activities != nil {
		for _, value := range user.Activities {
			if value.Name == activityName {
				msg = activityName + " is already added in the list of activities"
				echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
				return echoResp
			}
		}
	}

	//create an activity variable to add to the list in the database
	var activity = Activity{
		Name:         activityName,
		WhoseCurrent: 0,
	}
	spew.Dump(activity)

	//adds to db
	user.Activities = append(user.Activities, activity)
	col.Upsert(bson.M{"id": user.ID}, user)

	msg = "Added " + activityName + " to list of activities"
	fmt.Println(msg)
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)

	return echoResp
}

//handles removal of activity
//responds with error message if there is no activity by that name or another error
func removeActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	activityName, err := echoReq.GetSlotValue("Activity")
	if err != nil {
		log.Println("error")
		msg = "There was an error adding your activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	activityIndex := getActivityIndex(user.Activities, activityName)

	if activityIndex == -1 {
		msg = "There is no activity by that name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	user.Activities = append(user.Activities[:activityIndex], user.Activities[activityIndex+1:]...)

	updateUser(col, user)

	msg = "Removed " + activityName + " from the list of activities"

	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//gets index of a given activity array and name
func getActivityIndex(activities []Activity, activityName string) int {
	activityIndex := -1
	for index, activity := range activities {
		if activity.Name == activityName {
			activityIndex = index
			break
		}
	}
	return activityIndex
}

//handles remove person from activity intent
//responds with error if there is no activity or person by that name
func removePersonFromActivity(echoReq *alexa.EchoRequest, col *mgo.Collection, user *User) *alexa.EchoResponse {
	msg := ""

	activityName, err := echoReq.GetSlotValue("activity")
	personName, personErr := echoReq.GetSlotValue("person")

	if err != nil || personErr != nil {
		log.Println("error")
		msg = "There was an error adding them to your activity"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	activityIndex := -1
	for index, activity := range user.Activities {
		if activity.Name == activityName {
			activityIndex = index
			break
		}
	}

	if activityIndex == -1 {
		msg = "There is no activity by that name"
		echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
		return echoResp
	}

	spew.Dump(user.Activities[activityIndex].People)

	isPerson := false
	personIndex := 0
	for index, name := range user.Activities[activityIndex].People {
		if personName == name {
			isPerson = true
			personIndex = index
			break
		}
	}

	if isPerson == true {
		switch len(user.Activities[activityIndex].People) {
		case 1:
			user.Activities[activityIndex].People = append(user.Activities[activityIndex].People[:len(user.Activities[activityIndex].People)-1])

		default:
			user.Activities[activityIndex].People = append(user.Activities[activityIndex].People[:personIndex], user.Activities[activityIndex].People[personIndex+1:]...)
		}
	} else {
		msg = "There is no person by that name"
	}

	//update db
	updateUser(col, user)

	msg = "Removed " + personName + " from " + activityName
	echoResp := alexa.NewEchoResponse().OutputSpeech(msg).EndSession(false)
	return echoResp
}

//User struct with id and activities they are involved in
type User struct {
	ID         string     `json:"id"`
	Activities []Activity `json:"activities"`
}

//Activity struct that holds name, whose turn it currently is, and an array of people's names
type Activity struct {
	Name         string   `json:"name"`
	WhoseCurrent int      `json:"whoseCurrent"`
	People       []string `json:"people, omitempty"`
}
