package modeltest

import (
	"log"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/fyerfyer/chatroom/models"
)

func init() {
	go models.StartTest()
	time.Sleep(50 * time.Millisecond)
}

func hasUser(users []*models.User, user *models.User) bool {
	for _, u := range users {
		if u.Name == user.Name {
			return true
		}
	}

	return false
}

func TestBroadcastUserLogin(t *testing.T) {
	defer models.ClearUserListForTesting()

	user := &models.User{
		Name: "testing_user1",
	}
	models.TestBroadcaster.UserLogin(user)
	usersList := models.UserListForTesting()
	log.Println(len(usersList))
	if _, exist := usersList[user.Name]; !exist {
		t.Errorf("user %v should be in the map after login", user.Name)
	}

}

func TestBroadcateUserLogout(t *testing.T) {
	defer models.ClearUserListForTesting()

	user := &models.User{
		Name:           "testing_user2",
		MessageChannel: make(chan *models.Message),
	}
	models.TestBroadcaster.UserLogin(user)
	models.TestBroadcaster.UserLogout(user)
	usersList := models.UserListForTesting()
	if _, exist := usersList[user.Name]; exist {
		t.Errorf("user %v should not be in the map after logout", user.Name)
	}
}

func TestCheckUserCanLogin(t *testing.T) {
	defer models.ClearUserListForTesting()

	user := &models.User{
		Name: "testing_user3",
	}
	models.TestBroadcaster.UserLogin(user)
	canlogin := models.TestBroadcaster.CheckUserCanLogin("user4")
	cannotlogin := models.TestBroadcaster.CheckUserCanLogin(user.Name)
	if !canlogin {
		t.Errorf("new user should be allowed to login")
	}

	if cannotlogin {
		t.Errorf("login user should not be allowed to login")
	}
}

func TestGetUserList(t *testing.T) {
	defer models.ClearUserListForTesting()

	var users []*models.User
	for i := 0; i < 4; i++ {
		user := &models.User{
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *models.Message),
		}

		models.TestBroadcaster.UserLogin(user)
		users = append(users, user)
	}

	usersList := models.TestBroadcaster.GetUserList()
	if len(usersList) != 4 {
		t.Error("there should be 4 users in the user list")
	}
	for i := 0; i < 4; i++ {
		if !hasUser(usersList, users[i]) {
			t.Errorf("user %v should be in the user list", i)
		}
	}
}

func TestBroadcastMessage(t *testing.T) {
	defer models.ClearUserListForTesting()

	var users []*models.User
	for i := 0; i < 4; i++ {
		user := &models.User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *models.Message, 32),
		}
		users = append(users, user)
		models.LoginUserWithoutSendingMessage(user)
	}

	go models.TestBroadcaster.Broadcast(models.NewWelcomeMsg(users[0]))
	time.Sleep(50 * time.Millisecond)

	for i := 1; i < 4; i++ {
		if len(users[i].MessageChannel) == 0 {
			t.Errorf("user%v should have received a message", i)
		} else {
			for len(users[i].MessageChannel) > 0 {
				t.Log(<-users[i].MessageChannel)
			}
		}
	}
}

func TestLoginBroadcast(t *testing.T) {
	defer models.ClearUserListForTesting()

	var users []*models.User
	for i := 0; i < 3; i++ {
		user := &models.User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *models.Message, 32),
		}
		users = append(users, user)
		models.LoginUserWithoutSendingMessage(user)
	}

	user := &models.User{
		ID:             3,
		Name:           "testing_user3",
		MessageChannel: make(chan *models.Message, 32),
	}

	go models.TestBroadcaster.UserLogin(user)
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if len(users[i].MessageChannel) == 0 {
			t.Errorf("user%v should have received a message", i)
			return
		} else {
			for len(users[i].MessageChannel) > 0 {
				msg := <-users[i].MessageChannel
				if msg.Type != models.MsgTypeUserLogin {
					t.Error("the type of login message should be UserLogin")
					return
				} else {
					t.Log(msg)
				}
			}
		}
	}
}

func TestLogoutBroadcast(t *testing.T) {
	defer models.ClearUserListForTesting()

	var users []*models.User
	for i := 0; i < 4; i++ {
		user := &models.User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *models.Message, 32),
		}
		users = append(users, user)
		models.LoginUserWithoutSendingMessage(user)
	}

	go models.TestBroadcaster.UserLogout(users[3])
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if len(users[i].MessageChannel) == 0 {
			t.Errorf("user%v should have received a message", i)
			return
		} else {
			for len(users[i].MessageChannel) > 0 {
				msg := <-users[i].MessageChannel
				if msg.Type != models.MsgTypeUserLogout {
					t.Error("the type of logout message should be UserLogout")
					return
				} else {
					t.Log(msg)
				}
			}
		}
	}
}

func TestLoginConcurrency(t *testing.T) {
	defer models.ClearUserListForTesting()
	var wg sync.WaitGroup
	var boolCheck = make(map[bool]int)

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(name string) {
			defer wg.Done()
			user := &models.User{
				Name:           name,
				MessageChannel: make(chan *models.Message, 32),
			}

			var boolValue bool
			if boolValue = models.TestBroadcaster.CheckUserCanLogin(name); boolValue {
				models.TestBroadcaster.UserLogin(user)
				t.Log("successfully log in!")
			}

			boolCheck[boolValue]++
		}("testing_user0")
	}

	wg.Wait()

	if boolCheck[true] == 0 {
		t.Error("there should be at least one user successfully login")
		return
	}

	if boolCheck[true] > 1 {
		t.Error("there should not be more than one user successfully login")
		return
	}
}

func TestLogoutConcurrency(t *testing.T) {
	defer models.ClearUserListForTesting()
	var wg sync.WaitGroup
	var boolCheck = make(map[bool]int)
	models.LoginUserWithoutSendingMessage(&models.User{Name: "testing_user0", MessageChannel: make(chan *models.Message)})

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(name string) {
			defer wg.Done()

			var boolValue bool
			if boolValue = models.TestBroadcaster.CheckUserCanLogout(name); boolValue {
				models.TestBroadcaster.UserLogout(&models.User{Name: name, MessageChannel: make(chan *models.Message)})
				t.Log("successfully log out!")
			}

			boolCheck[boolValue]++
		}("testing_user0")
	}

	wg.Wait()

	if boolCheck[true] == 0 {
		t.Error("there should be at least one user successfully logout")
		return
	}

	if boolCheck[true] > 1 {
		t.Error("there should not be more than one user successfully logout")
		return
	}
}
