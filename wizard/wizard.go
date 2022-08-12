package wizard

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"goblog/auth"
	"goblog/tools"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type Wizard struct {
	Version string
}

// New constructs an Admin API
func New(version string) Wizard {
	wizard := Wizard{version}
	return wizard
}

func (w Wizard) Landing(c *gin.Context) {
	c.HTML(http.StatusOK, "wizard.html", gin.H{
		"version": w.Version,
		"title":   "GoBlog Install Wizard",
	})
}

func (w Wizard) SaveToken(c *gin.Context) {
	sqlite := c.Query("sqlite")
	if sqlite != "on" {
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Only support sqlite at the moment",
		})
		return
	}
	client_id := c.Query("client_id")
	if client_id == "" {
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Client ID must not be empty",
		})
		return
	}
	client_secret := c.Query("client_secret")
	if client_secret == "" {
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Client Secret must not be empty",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("client_id", client_id)
	session.Set("client_secret", client_secret)
	err := session.Save()
	if err != nil {
		log.Println("Can't save client_id and client_secret")
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Can't save client_id and client_secret",
		})
		return
	}

	c.Redirect(http.StatusFound, "https://github.com/login/oauth/authorize?client_id="+client_id)
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (w Wizard) LoginCode(c *gin.Context) {
	code := c.Query("code")
	log.Println("LOGIN CODE: " + code)

	session := sessions.Default(c)
	clientId, ok := session.Get("client_id").(string)
	if !ok {
		log.Println("Can't retrieve client_id")
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Couldn't retrieve the client_id",
		})
		return
	}

	clientSecret, ok := session.Get("client_secret").(string)
	if !ok {
		log.Fatal("Can't retrieve client_secret")
	}

	formData := url.Values{
		"client_id":     {clientId},
		"client_secret": {clientSecret},
		"code":          {code},
	}

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(formData.Encode()))
	if err != nil {
		log.Println("Error requesting access token from github: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error requesting access token from github: " + err.Error(),
		})
		return
	}

	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error requesting access token from github: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error requesting access token from github: " + err.Error(),
		})
		return
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error requesting access token from github: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error requesting access token from github: " + err.Error(),
		})
		return
	}

	bodyString := string(bodyBytes)
	fmt.Println("post:\n", bodyString) //todo: remove - just for debugging

	if resp.StatusCode != http.StatusOK {
		log.Println("Error requesting access token from github: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error requesting access token from github: " + err.Error(),
		})
		return
	}

	tokenResponse := &AccessTokenResponse{}
	json.Unmarshal(bodyBytes, &tokenResponse)

	err = w.createEnvFile(clientId, clientSecret)
	if err != nil {
		log.Println("Error writing the .env file to save settings: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error writing the .env file to save settings: " + err.Error(),
		})
		return
	}

	session.Delete("client_id")
	session.Delete("client_secret")
	err = session.Save()
	if err != nil {
		log.Println("Can't clearing the client_id and client_secrete from cookies: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Can't clearing the client_id and client_secrete from cookies: " + err.Error(),
		})
		return
	}

	err = w.createDbFile(tokenResponse.AccessToken)
	if err != nil {
		log.Println("Error creating the db: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error creating the db: " + err.Error(),
		})
		return
	}

	err = w.killWizardServer()
	if err != nil {
		log.Println("Error killing the wizard server: " + err.Error())
		c.HTML(http.StatusOK, "wizard.html", gin.H{
			"version": w.Version,
			"errors":  "Error killing the wizard server: " + err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "wizard_success.html", gin.H{
		"version": w.Version,
	})
}

func (w Wizard) killWizardServer() error {
	file, err := os.Open("/tmp/goblog.pid")
	if err != nil {
		return errors.New("couldn't open the /tmp/goblog.pid file: " + err.Error())
	}
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		pid_str := scanner.Text()
		err = file.Close()
		if err != nil {
			return errors.New("couldn't close the /tmp/goblog.pid file: " + err.Error())
		}
		err = os.Remove("/tmp/goblog.pid")
		if err != nil {
			return errors.New("couldn't remove the /tmp/goblog.pid file: " + err.Error())
		}
		pid, err := strconv.Atoi(pid_str)
		if err != nil {
			return errors.New("couldn't parse the pid from the /tmp/goblog.pid file: " + err.Error())
		}
		log.Println("Successfully completed the wizard, shutting the server down now so the main server can start")
		err = syscall.Kill(pid, syscall.SIGINT)
		if err != nil {
			return errors.New("couldn't kill the wizard server at pid: " + pid_str + ": " + err.Error())
		}

	} else {
		return errors.New("couldn't read a line from the /tmp/goblog.pid file")
	}
	return nil
}

// attempts to create the .env file and fill it with the required config
func (w Wizard) createEnvFile(clientID string, clientSecret string) error {
	f, err := os.Create(".env")
	if err != nil {
		return errors.New("coudln't create the .env file: " + err.Error())
	}
	_, err = f.WriteString("database=sqlite\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	_, err = f.WriteString("sqlite_db=dev.db\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	_, err = f.WriteString("\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	_, err = f.WriteString("# github auth credentials\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	_, err = f.WriteString("client_id=" + clientID + "\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	_, err = f.WriteString("client_secret=" + clientSecret + "\n")
	if err != nil {
		return errors.New("coudln't write to the .env file: " + err.Error())
	}
	err = f.Sync()
	if err != nil {
		return errors.New("coudln't sync the .env file: " + err.Error())
	}
	err = f.Close()
	if err != nil {
		return errors.New("coudln't close the .env file: " + err.Error())
	}

	return nil
}

// attempts to create the db and setup the schema
func (w Wizard) createDbFile(accessToken string) error {
	db, err := gorm.Open(sqlite.Open("dev.db"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return err
	}
	log.Println("opened sqlite db")
	tools.Migrate(db)
	_auth := auth.New(db, w.Version)

	user, err := _auth.RequestUser(accessToken)
	if err != nil {
		return errors.New("couldn't get user data from github")
	} else {
		fmt.Printf("GOT USER: %+v\n", user)
	}

	adminUser := auth.AdminUser{BlogUserID: user.ID, BlogUser: *user}
	result := db.Create(&adminUser)
	if result.Error != nil || result.RowsAffected == 0 {
		return errors.New("Error creating admin user: " + result.Error.Error())
	}

	return nil
}
