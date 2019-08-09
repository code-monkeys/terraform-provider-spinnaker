package spinnaker

import (
	"strings"

	"github.com/armory-io/terraform-provider-spinnaker/spinnaker/api"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceApplication() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"application": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"instance_port": {
				Type:         schema.TypeInt,
				Required:     false,
				Optional:     true,
				Default:      80,
				ValidateFunc: validation.IntBetween(1, 65535),
			},
			"permissions": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"READ", "EXEC", "WRITE"}, false),
				},
			},
		},
		Create: resourceApplicationCreate,
		Read:   resourceApplicationRead,
		Update: resourceApplicationUpdate,
		Delete: resourceApplicationDelete,
		Exists: resourceApplicationExists,
	}
}

// application represents the Gate API schema
//
// HINT: to extend this schema have a look at the output
// of the spin (https://github.com/spinnaker/spin)
// application get command.
type application struct {
	Name         string              `json:"name"`
	Email        string              `json:"email"`
	InstancePort int                 `json:"instancePort"`
	Permissions  map[string][]string `json:"permissions"`
}

// applicationRead represents the Gate API schema of an application
// get request. The relevenat part of the schema is identical with
// the application struct, it's just wrapped in an attributes field.
type applicationRead struct {
	Name       string       `json:"name"`
	Attributes *application `json:"attributes"`
}

func resourceApplicationCreate(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(gateConfig)
	client := clientConfig.client

	app := applicationFromResource(data)
	if err := api.CreateApplication(client, app.Name, app); err != nil {
		return err
	}

	return resourceApplicationRead(data, meta)
}

func resourceApplicationRead(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(gateConfig)
	client := clientConfig.client

	applicationName := data.Get("application").(string)
	app := &applicationRead{}
	if err := api.GetApplication(client, applicationName, app); err != nil {
		return err
	}

	return readApplication(data, app)
}

func resourceApplicationUpdate(data *schema.ResourceData, meta interface{}) error {
	// the application update in spinnaker is an simple upsert
	clientConfig := meta.(gateConfig)
	client := clientConfig.client

	app := applicationFromResource(data)
	if err := api.CreateApplication(client, app.Name, app); err != nil {
		return err
	}

	return resourceApplicationRead(data, meta)
}

func resourceApplicationDelete(data *schema.ResourceData, meta interface{}) error {
	clientConfig := meta.(gateConfig)
	client := clientConfig.client
	applicationName := data.Get("application").(string)

	return api.DeleteAppliation(client, applicationName)
}

func resourceApplicationExists(data *schema.ResourceData, meta interface{}) (bool, error) {
	clientConfig := meta.(gateConfig)
	client := clientConfig.client
	applicationName := data.Get("application").(string)

	var app applicationRead
	if err := api.GetApplication(client, applicationName, &app); err != nil {
		errmsg := err.Error()
		if strings.Contains(errmsg, "not found") {
			return false, nil
		}
		return false, err
	}

	if app.Name == "" {
		return false, nil
	}

	return true, nil
}

func applicationFromResource(data *schema.ResourceData) *application {
	app := &application{
		Name:         data.Get("application").(string),
		Email:        data.Get("email").(string),
		InstancePort: data.Get("instance_port").(int),
		Permissions:  make(map[string][]string),
	}

	for team, permI := range data.Get("permissions").(map[string]interface{}) {
		perm := permI.(string)
		app.Permissions[perm] = append(app.Permissions[perm], team)
	}

	return app
}

func readApplication(data *schema.ResourceData, application *applicationRead) error {
	data.SetId(application.Name)
	data.Set("name", application.Name)
	data.Set("email", application.Attributes.Email)
	data.Set("instance_port", application.Attributes.InstancePort)

	perms := make(map[string][]string)
	for perm, teams := range application.Attributes.Permissions {
		for _, team := range teams {
			perms[team] = append(perms[team], perm)
		}
	}
	data.Set("permissions", perms)
	return nil
}
