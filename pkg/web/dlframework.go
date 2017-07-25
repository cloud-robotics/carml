package web

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	openapierrors "github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	runtime "github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/k0kubun/pp"
	"github.com/pkg/errors"

	"fmt"

	"github.com/rai-project/config"
	dlframework "github.com/rai-project/dlframework"
	"github.com/rai-project/dlframework/web/models"
	restapi "github.com/rai-project/dlframework/web/restapi"
	"github.com/rai-project/dlframework/web/restapi/operations"
	"github.com/rai-project/dlframework/web/restapi/operations/predictor"
	"github.com/rai-project/dlframework/web/restapi/operations/registry"
	kv "github.com/rai-project/registry"
)

type apiError struct {
	Name    string
	Message error
	Code    int
}

func (e apiError) Error() string {
	return e.Message.Error()
}

func (e apiError) MarshalJSON() ([]byte, error) {
	var stack string
	name := fmt.Sprintf("\"name\": \"%s\"", e.Name)
	message := fmt.Sprintf("\"message\": \"%s\"", e.Message.Error())
	code := fmt.Sprintf("\"code\": %d", e.Code)
	stackData := strings.Split(fmt.Sprintf("%+v", e.Message), "\n")
	bts, err := json.Marshal(stackData)
	if err != nil {
		stack = fmt.Sprintf("\"stack\": []")
	} else {
		stack = fmt.Sprintf("\"stack\": %s", string(bts))
	}
	res := fmt.Sprintf("{%s, %s, %s, %s}", name, message, code, stack)
	return []byte(res), nil
}

func getDlframeworkHandler() (http.Handler, error) {
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		return nil, err
	}
	api := operations.NewDlframeworkAPI(swaggerSpec)

	api.ServeError = openapierrors.ServeError
	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	makeError := func(code int, name string, message error) error {
		return apiError{Code: code, Name: name, Message: message}
	}

	api.RegistryGetFrameworkManifestHandler = registry.GetFrameworkManifestHandlerFunc(func(params registry.GetFrameworkManifestParams) middleware.Responder {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			fn := params.FrameworkName
			fv := params.FrameworkVersion

			if fv == nil {
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						errors.New("invalid RegistryGetFrameworkManifestHandler framework version cannot be empty"),
					),
				)
				return
			}

			rgs, err := kv.New()
			if err != nil {
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
				rw.WriteHeader(http.StatusBadRequest)
				return
			}
			defer rgs.Close()

			key := path.Join(config.App.Name, "registry", fn, *fv, "info")
			ok, err := rgs.Exists(key)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
				return
			}
			if !ok {
				registry.NewGetFrameworkManifestOK().
					WithPayload(&models.DlframeworkFrameworkManifest{}).
					WriteResponse(rw, producer)
				return
			}
			e, err := rgs.Get(key)
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
				return
			}
			framework := new(dlframework.FrameworkManifest)
			if err := framework.Unmarshal(e.Value); err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
				return
			}

			container := map[string]models.DlframeworkContainerHardware{}
			for k, v := range framework.GetContainer() {
				container[k] = models.DlframeworkContainerHardware{
					CPU: v.GetCpu(),
					Gpu: v.GetGpu(),
				}
			}

			res := &models.DlframeworkFrameworkManifest{
				Container: container,
				Name:      framework.GetName(),
				Version:   framework.GetVersion(),
			}
			pp.Println(res)

			registry.NewGetFrameworkManifestOK().
				WithPayload(res).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetFrameworkManifestsHandler = registry.GetFrameworkManifestsHandlerFunc(func(params registry.GetFrameworkManifestsParams) middleware.Responder {

		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			rgs, err := kv.New()
			if err != nil {
				panic(err)
			}
			defer rgs.Close()

			manifests := []*models.DlframeworkFrameworkManifest{}

			dirs := []string{path.Join(config.App.Name, "registry")}
			for {
				if len(dirs) == 0 {
					break
				}
				var dir string
				dir, dirs = dirs[0], dirs[1:]
				lst, err := rgs.List(dir)
				if err != nil {
					continue
				}
				for _, e := range lst {
					if e.Value == nil || len(e.Value) == 0 {
						dirs = append(dirs, e.Key)
						continue
					}
					framework := new(dlframework.FrameworkManifest)
					if err := framework.Unmarshal(e.Value); err != nil {
						continue
					}
					container := map[string]models.DlframeworkContainerHardware{}
					for k, v := range framework.GetContainer() {
						if v == nil {
							continue
						}
						container[k] = models.DlframeworkContainerHardware{
							CPU: v.GetCpu(),
							Gpu: v.GetGpu(),
						}
					}
					manifests = append(manifests, &models.DlframeworkFrameworkManifest{
						Container: container,
						Name:      framework.GetName(),
						Version:   framework.GetVersion(),
					})
				}
			}

			registry.NewGetFrameworkManifestsOK().
				WithPayload(&models.DlframeworkGetFrameworkManifestsResponse{
					Manifests: manifests,
				}).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetFrameworkModelManifestHandler = registry.GetFrameworkModelManifestHandlerFunc(func(params registry.GetFrameworkModelManifestParams) middleware.Responder {
		return middleware.NotImplemented("operation registry.GetFrameworkModelManifest has not yet been implemented")
	})
	api.RegistryGetFrameworkModelsHandler = registry.GetFrameworkModelsHandlerFunc(func(params registry.GetFrameworkModelsParams) middleware.Responder {
		return middleware.NotImplemented("operation registry.GetFrameworkModels has not yet been implemented")
	})
	api.RegistryGetModelManifestHandler = registry.GetModelManifestHandlerFunc(func(params registry.GetModelManifestParams) middleware.Responder {
		return middleware.NotImplemented("operation registry.GetModelManifest has not yet been implemented")
	})
	api.RegistryGetModelManifestsHandler = registry.GetModelManifestsHandlerFunc(func(params registry.GetModelManifestsParams) middleware.Responder {
		return middleware.NotImplemented("operation registry.GetModelManifests has not yet been implemented")
	})
	api.PredictorPredictHandler = predictor.PredictHandlerFunc(func(params predictor.PredictParams) middleware.Responder {
		return middleware.NotImplemented("operation predictor.Predict has not yet been implemented")
	})

	api.ServerShutdown = func() {}
	api.Logger = log.Debugf

	handler := api.Serve(nil)

	return handler, nil
}
