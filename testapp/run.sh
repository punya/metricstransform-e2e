#!/bin/sh

set +o verbose
cd `dirname $0`
go run . $@
