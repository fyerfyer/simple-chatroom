package models

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func clearUserListForTesting() {
	Broadcaster.users = make(map[string]*User)
}

func loginUserWithoutSendingMessage(user *User) {
	Broadcaster.users[user.Name] = user
}

func init() {
	go Broadcaster.Start()
	time.Sleep(50 * time.Millisecond)
}

func hasUser(users []*User, user *User) bool {
	for _, u := range users {
		if u.Name == user.Name {
			return true
		}
	}

	return false
}

func TestBroadcastUserLogin(t *testing.T) {
	defer clearUserListForTesting()

	user := &User{
		Name: "testing_user1",
	}

	Broadcaster.UserLogin(user)
	if _, exist := Broadcaster.users[user.Name]; !exist {
		t.Errorf("user %v should be in the map after login", user.Name)
	}

}

func TestBroadcateUserLogout(t *testing.T) {
	defer clearUserListForTesting()

	user := &User{
		Name:           "testing_user2",
		MessageChannel: make(chan *Message),
	}

	Broadcaster.UserLogin(user)
	Broadcaster.UserLogout(user)
	if _, exist := Broadcaster.users[user.Name]; exist {
		t.Errorf("user %v should not be in the map after logout", user.Name)
	}
}

func TestCheckUserCanLogin(t *testing.T) {
	defer clearUserListForTesting()

	user := &User{
		Name: "testing_user3",
	}
	Broadcaster.UserLogin(user)
	canlogin := Broadcaster.CheckUserCanLogin("user4")
	cannotlogin := Broadcaster.CheckUserCanLogin(user.Name)
	if !canlogin {
		t.Errorf("new user should be allowed to login")
	}

	if cannotlogin {
		t.Errorf("login user should not be allowed to login")
	}
}

func TestGetUserList(t *testing.T) {
	defer clearUserListForTesting()

	var users []*User
	for i := 0; i < 4; i++ {
		user := &User{
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *Message),
		}

		Broadcaster.UserLogin(user)
		users = append(users, user)
	}

	usersList := Broadcaster.GetUserList()
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
	defer clearUserListForTesting()

	var users []*User
	for i := 0; i < 4; i++ {
		user := &User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *Message, 32),
		}
		users = append(users, user)
		loginUserWithoutSendingMessage(user)
	}

	go Broadcaster.Broadcast(NewWelcomeMsg(users[0]))
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
	defer clearUserListForTesting()

	var users []*User
	for i := 0; i < 3; i++ {
		user := &User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *Message, 32),
		}
		users = append(users, user)
		loginUserWithoutSendingMessage(user)
	}

	user := &User{
		ID:             3,
		Name:           "testing_user3",
		MessageChannel: make(chan *Message, 32),
	}

	go Broadcaster.UserLogin(user)
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if len(users[i].MessageChannel) == 0 {
			t.Errorf("user%v should have received a message", i)
			return
		} else {
			for len(users[i].MessageChannel) > 0 {
				msg := <-users[i].MessageChannel
				if msg.Type != MsgTypeUserLogin {
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
	defer clearUserListForTesting()

	var users []*User
	for i := 0; i < 4; i++ {
		user := &User{
			ID:             i,
			Name:           "testing_user" + strconv.Itoa(i),
			MessageChannel: make(chan *Message, 32),
		}
		users = append(users, user)
		loginUserWithoutSendingMessage(user)
	}

	go Broadcaster.UserLogout(users[3])
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 3; i++ {
		if len(users[i].MessageChannel) == 0 {
			t.Errorf("user%v should have received a message", i)
			return
		} else {
			for len(users[i].MessageChannel) > 0 {
				msg := <-users[i].MessageChannel
				if msg.Type != MsgTypeUserLogout {
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
	defer clearUserListForTesting()
	var wg sync.WaitGroup
	var boolCheck = make(map[bool]int)

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(name string) {
			defer wg.Done()
			user := &User{
				Name:           name,
				MessageChannel: make(chan *Message, 32),
			}

			var boolValue bool
			if boolValue = Broadcaster.CheckUserCanLogin(name); boolValue {
				Broadcaster.UserLogin(user)
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
	defer clearUserListForTesting()
	var wg sync.WaitGroup
	var boolCheck = make(map[bool]int)
	loginUserWithoutSendingMessage(&User{Name: "testing_user0", MessageChannel: make(chan *Message)})

	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func(name string) {
			defer wg.Done()

			var boolValue bool
			if boolValue = Broadcaster.CheckUserCanLogout(name); boolValue {
				Broadcaster.UserLogout(&User{Name: name, MessageChannel: make(chan *Message)})
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
