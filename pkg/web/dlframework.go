package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	openapierrors "github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	runtime "github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/gogo/protobuf/jsonpb"
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
	"github.com/ulule/deepcopier"
)

var (
	DefaultUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: false}
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

	unmarshaler := DefaultUnmarshaler

	getFrameworks := func() ([]*models.DlframeworkFrameworkManifest, error) {
		rgs, err := kv.New()
		if err != nil {
			return nil, err
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
				registryValue := e.Value
				framework := new(dlframework.FrameworkManifest)
				if err := unmarshaler.Unmarshal(bytes.NewBuffer(registryValue), framework); err != nil {
					continue
				}
				res := new(models.DlframeworkFrameworkManifest)
				if err := deepcopier.Copy(framework).To(res); err != nil {
					continue
				}
				manifests = append(manifests, res)
			}
		}
		return manifests, nil
	}

	api.RegistryGetFrameworkManifestHandler = registry.GetFrameworkManifestHandlerFunc(func(params registry.GetFrameworkManifestParams) middleware.Responder {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			var err error
			defer func() {
				if err == nil {
					return
				}
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
			}()

			fn := strings.ToLower(params.FrameworkName)
			fv := params.FrameworkVersion
			if fv == nil {
				err = errors.New("invalid RegistryGetFrameworkManifestHandler framework version cannot be empty")
				return
			}

			rgs, err := kv.New()
			if err != nil {
				return
			}
			defer rgs.Close()

			key := path.Join(config.App.Name, "registry", fn, strings.ToLower(*fv), "info")
			ok, err := rgs.Exists(key)
			if err != nil {
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
				return
			}
			framework := new(dlframework.FrameworkManifest)
			registryValue := e.Value
			err = unmarshaler.Unmarshal(bytes.NewBuffer(registryValue), framework)
			if err != nil {
				return
			}

			res := new(models.DlframeworkFrameworkManifest)
			if err := deepcopier.Copy(framework).To(res); err != nil {
				return
			}

			registry.NewGetFrameworkManifestOK().
				WithPayload(res).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetFrameworkManifestsHandler = registry.GetFrameworkManifestsHandlerFunc(func(params registry.GetFrameworkManifestsParams) middleware.Responder {

		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			frameworks, err := getFrameworks()
			if err != nil {
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkModelManifest",
						err,
					),
				)
				return
			}

			registry.NewGetFrameworkManifestsOK().
				WithPayload(&models.DlframeworkGetFrameworkManifestsResponse{
					Manifests: frameworks,
				}).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetFrameworkModelManifestHandler = registry.GetFrameworkModelManifestHandlerFunc(func(params registry.GetFrameworkModelManifestParams) middleware.Responder {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			var err error
			defer func() {
				if err == nil {
					return
				}
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
			}()

			fn := strings.ToLower(params.FrameworkName)
			fv := "latest"
			if params.FrameworkVersion != nil {
				fv = strings.ToLower(*params.FrameworkVersion)
			}
			mn := strings.ToLower(params.ModelName)
			mv := "latest"
			if params.ModelVersion != nil {
				mv = strings.ToLower(*params.ModelVersion)
			}

			if fv == "" {
				err = errors.New("invalid RegistryGetFrameworkModelManifestHandler framework version cannot be empty")
				return
			}
			if mv == "" {
				err = errors.New("invalid RegistryGetFrameworkModelManifestHandler model version cannot be empty")
				return
			}

			rgs, err := kv.New()
			if err != nil {
				return
			}
			defer rgs.Close()

			key := path.Join(config.App.Name, "registry", fn, fv, mn, mv, "info")
			ok, err := rgs.Exists(key)
			if err != nil {
				return
			}
			if !ok {
				registry.NewGetFrameworkModelManifestOK().
					WithPayload(&models.DlframeworkModelManifest{}).
					WriteResponse(rw, producer)
				return
			}
			e, err := rgs.Get(key)
			if err != nil {
				return
			}
			registryValue := e.Value
			model := new(dlframework.ModelManifest)
			err = unmarshaler.Unmarshal(bytes.NewBuffer(registryValue), model)
			if err != nil {
				return
			}

			res := new(models.DlframeworkModelManifest)
			err = deepcopier.Copy(model).To(res)
			if err != nil {
				return
			}

			registry.NewGetFrameworkModelManifestOK().
				WithPayload(res).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetFrameworkModelsHandler = registry.GetFrameworkModelsHandlerFunc(func(params registry.GetFrameworkModelsParams) middleware.Responder {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			var err error
			defer func() {
				if err == nil {
					return
				}
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
			}()

			frameworks, err := getFrameworks()
			if err != nil {
				return
			}

			rgs, err := kv.New()
			if err != nil {
				return
			}
			defer rgs.Close()

			manifests := []*models.DlframeworkModelManifest{}

			for _, framework := range frameworks {
				basePath := path.Join(config.App.Name, "registry", framework.Name, framework.Version)
				dirs := []string{basePath}
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
						if e.Key == path.Join(basePath, "info") {
							continue
						}
						if e.Value == nil || len(e.Value) == 0 {
							dirs = append(dirs, e.Key)
							continue
						}

						registryValue := e.Value
						model := new(dlframework.ModelManifest)
						err = unmarshaler.Unmarshal(bytes.NewBuffer(registryValue), model)
						if err != nil {
							continue
						}

						res := new(models.DlframeworkModelManifest)
						err = deepcopier.Copy(model).To(res)
						if err != nil {
							continue
						}

						manifests = append(manifests, res)
					}
				}
			}

			registry.NewGetFrameworkModelsOK().
				WithPayload(&models.DlframeworkGetModelManifestsResponse{
					Manifests: manifests,
				}).
				WriteResponse(rw, producer)
		})
	})
	api.RegistryGetModelManifestHandler = registry.GetModelManifestHandlerFunc(func(params registry.GetModelManifestParams) middleware.Responder {
		return middleware.ResponderFunc(func(rw http.ResponseWriter, producer runtime.Producer) {
			var err error
			defer func() {
				if err == nil {
					return
				}
				rw.WriteHeader(http.StatusBadRequest)
				producer.Produce(rw,
					makeError(
						http.StatusBadRequest,
						"GetFrameworkManifest",
						err,
					),
				)
			}()

			modelName := strings.ToLower(params.ModelName)
			if modelName == "" {
				err = errors.New("invalid empty model name")
				return
			}
			if params.ModelVersion == nil {
				err = errors.New("invalid  model version")
				return
			}
			modelVersion := strings.ToLower(*params.ModelVersion)
			if modelVersion == "" {
				err = errors.New("invalid empty model version")
				return
			}

			frameworks, err := getFrameworks()
			if err != nil {
				return
			}

			rgs, err := kv.New()
			if err != nil {
				return
			}
			defer rgs.Close()

			for _, framework := range frameworks {
				frameworkName, frameworkVersion := strings.ToLower(framework.Name), strings.ToLower(framework.Version)
				key := path.Join(config.App.Name, "registry", frameworkName, frameworkVersion, modelName, modelVersion, "info")
				ok, err := rgs.Exists(key)
				if err != nil {
					return
				}
				if !ok {
					continue
				}
				e, err := rgs.Get(key)
				if err != nil {
					return
				}
				registryValue := e.Value
				model := new(dlframework.ModelManifest)
				err = unmarshaler.Unmarshal(bytes.NewBuffer(registryValue), model)
				if err != nil {
					continue
				}

				res := new(models.DlframeworkModelManifest)
				err = deepcopier.Copy(model).To(res)
				if err != nil {
					continue
				}

				registry.NewGetModelManifestOK().
					WithPayload(res).
					WriteResponse(rw, producer)
				return
			}
			registry.NewGetFrameworkModelManifestOK().
				WithPayload(&models.DlframeworkModelManifest{}).
				WriteResponse(rw, producer)
			return

		})
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
