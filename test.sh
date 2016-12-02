#!/bin/bash

WD=$(pwd)
WORKDIR=tmp
TFVERSION=0.7.11
if [[ $OSTYPE =~ darwin ]]; then
  TFARCH=darwin_amd64
else
  TFARCH=linux_amd64
fi
TFURL="https://releases.hashicorp.com/terraform/${TFVERSION}/terraform_${TFVERSION}_${TFARCH}.zip"

if ! [[ -d $WORKDIR ]]; then
  mkdir $WORKDIR
fi
cd $WORKDIR

if ! [[ -e terraform ]]; then
  echo "downloading terraform"
  curl -s $TFURL -o terraform_${TFVERSION}_${TFARCH}.zip
  if [[ $? -ne 0 ]]; then
    echo "failed to download terraform"
    exit 1
  fi
  unzip terraform_${TFVERSION}_${TFARCH}.zip
fi

# check terraform version
VCHECK="$(./terraform version | head -n1 | awk '{print $2}')"
if [[ $VCHECK != "v${TFVERSION}" ]]; then
  echo "Terraform version (${VCHECK}) doesn't match ${TFVERSION}"
  exit 1
fi

if ! [[ -e $CALICOBIN ]]; then
  echo "downloading calicoctl"
  go get -v github.com/bolcom/calico-containers/calicoctl
  if [[ $? -ne 0 ]]; then
    echo "Failed to download calicoctl"
    exit 1
  fi
fi

# check calicoctl
CALICOBIN="${GOPATH}/bin/calicoctl"
if ! $CALICOBIN version &>/dev/null; then
  echo "Built calicoctl doesn't work as expected"
  exit 1
fi
cd "$WD"

if [[ "$DEBUG" != "true" ]]; then
  echo "Downloading GO dependencies"
  go get -v
  if [[ $? -ne 0 ]]; then
    echo "Failed to download all dependencies"
    exit 1
  fi
fi

echo "Building terraform-provider-calico:"
go build -v
if [[ $? -ne 0 ]]; then
  echo "Failed to build terraform-provider-calico"
  exit 1
fi
cp terraform-provider-calico $WORKDIR
cp testing/* $WORKDIR
cd $WORKDIR

if ! grep "${WD}/${WORKDIR}/test/terraform-provider-calico" ~/.terraformrc 2>&1 > /dev/null; then
  echo
  echo "You'll have to change your ~/.terraformrc file to include this"
  echo "if you want to continue running these tests:"
  echo
  echo "providers {"
  echo "  calico = \"${WD}/${WORKDIR}/test/terraform-provider-calico\""
  echo "}"
  exit 1
fi

# cleanup just in case
docker stop $(docker-compose ps -q) &>/dev/null
docker-compose kill &>/dev/null
docker-compose rm &>/dev/null

echo "Setting up Etcd"
docker-compose run -p 2379:2379 -d etcd >/dev/null

if [[ $OSTYPE =~ darwin ]]; then
  if [ -z "$DOCKER_HOST" ]; then
    echo "Missing DOCKER_HOST environment key"
    exit 1
  fi
  hostport=${DOCKER_HOST##*//}
  ETCD_AUTHORITY="${hostport%%:*}:2379"
else
  ETCD_AUTHORITY="$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $(docker-compose ps -q)):2379"
fi

if [[ "$ETCD_AUTHORITY" == "" ]]; then
  echo "Failed to get Etcd endpoint"
  exit 1
fi
echo "(ETCD_AUTHORITY=$ETCD_AUTHORITY)"

sleep 5s

rm -rf test; mkdir test
sed "s/PLACEHOLDER/${ETCD_AUTHORITY}/" provider.tf > test/provider.tf

cp terraform test/
cp terraform-provider-calico test/
cp "$CALICOBIN" test/

echo
echo "Testing:"
RESOURCES="${TESTS:-nodes hostendpoints profiles workloadendpoints ippools bgppeers policies}"
for i in $RESOURCES; do
  tffile="${WD}/testing/test_${i}.tf"
  if [[ -e $tffile ]]; then
    cp "$tffile" test/
    cd test

    if [[ "$DEBUG" == "true" ]]; then
      export TF_LOG=DEBUG
    fi

    RES="$(./terraform apply)"
    if [[ $? -ne 0 ]]; then
      echo "${i} - FAILED"
      echo "$RES"
      echo "Failed to terraform apply (${tffile})"
      exit 1
    fi
    rm "test_${i}.tf"

    ETCD_AUTHORITY="$ETCD_AUTHORITY" ./calicoctl get $i -o yaml 1> test.yaml 2>calicoctl_debug.txt
    if [[ $? -ne 0 ]]; then
      echo "${i} - FAILED"
      echo "Failed to talk to Etcd at ${ETCD_AUTHORITY}"
      exit 1
    fi

    RES="$(diff test.yaml ${WD}/testing/test_${i}.yaml)"
    if [[ $? -ne 0 ]]; then
      echo "${i} - FAILED"
      echo "${RES}"
      echo "Expected ${i} yaml and that from testing/test_${i}.yaml do not match"
      echo "Full output from Etcd:"
      cat test.yaml
      exit 1
    else
      echo "${i} - OK"
      if [[ "$DEBUG" == "true" ]]; then
        cat test.yaml
        echo "Full output from Calicoctl:"
        cat calicoctl_debug.txt
        rm calicoctl_debug.txt
      fi
      cd ..
    fi
  else
    echo "${i} - Not implemented"
  fi
done
