## About Who’s Next

Who’s next is an Amazon Alexa application I developed to experiment with AWS (EC2/Alexa), GoLang, and MongoDB.

WhoseTurn.go is the code that interfaces with the Alexa Skill I created with the native Amazon developer console. This code is the handler for the Alexa Skill requests that are made when a user interacts with the skill. 

The application I developed tracks a list of tasks and the people who are taking turns performing each task. You can add “activities” i.e. tasks, and people who take turns doing the tasks. The main “intents” are add activity, list activities, add person to activity, who’s next for activity, and someone completed an activity. I used the Alexa developer console to add Intents and sample utterances. Amazon has good documentation on how to use the development console and examples. This was probably the easiest part of the project. 

The Alexa service uses JSON to send service requests. I found a GoLang package which implements the necessary validation, parsing, routing, and middleware for Alexa JSON objects. The package is called SkillServer and is written by Mike Flynn. The framework uses Gorrila/Mux to route HTTP requests to the appropriate request handlers. Negroni is the middleware package the provides wrappers for the request handlers. The SkillServer unpacks the JSON requests into nice easy to use objects and then calls the appropriate handler for each service request. It is left to the user to create handlers for each intent unique to the Alexa skill being implemented. I used MongoDB to persist user data including the activities list and people associated with each activity.
