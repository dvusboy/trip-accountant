NAME            := trip-accountant
REPO             = $(shell git config --get remote.origin.url 2>/dev/null | sed -e 's|^.\{1,\}github\.com[:/]\(.\{1,\}\)$$|\1|' -e 's/\.git$$//')
MAJOR           := $(shell git describe --always --long | sed -e 's/-[^-]\{1,\}$$//' | awk -F- '{ print $$1 }')
MINOR           := $(shell git describe --always --long | sed -e 's/-[^-]\{1,\}$$//' | awk -F- '{ print $$2 }')
VERSION         ?= $(MAJOR).$(MINOR)
COMMIT          := $(shell git rev-parse HEAD)
GO_VERSION      := 1.24
RELEASE         ?= 1
TAG              = dvusboy/$(NAME):$(VERSION)-$(RELEASE)
LOG              = build-$(VERSION).log
MARKER           = .image.done.$(VERSION)

.PHONY : default
default : $(MARKER)

$(MARKER) :
	[ -s "$@" ] && docker image rm `cat "$@"`; rm -f "$@"
	docker build --pull --rm \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg VERSION=$(VERSION) \
	--build-arg REPO=github.com/$(REPO) \
	--build-arg COMMIT=$(COMMIT) \
	--iidfile=$@ \
	--tag=$(TAG) . 2>&1 | tee $(LOG)
	[ -s "$@" ] || rm -f "$@"

.PHONY : push
push : $(MARKER)
	docker push $(TAG)

.PHONY : clean
clean :
	rm -f $(LOG) $(NAME)

.PHONY : clean-img
clean-img :
	[ -s "$(MARKER)" ] && docker image rm `cat $(MARKER)`; rm -f $(MARKER)

.PHONY : clean-all
clean-all : clean clean-img

.PHONY : go-test
go-test :
	go test -v ./...

.PHONY : api-test
api-test : testAPI.sh $(MARKER)
	./testAPI.sh $(TAG)

.PHONY : test
test : go-test api-test
