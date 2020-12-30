#! /bin/bash

RUBY=~/.rvm/environments/ruby-2.7.2

if [[ ! -f $RUBY ]] ; then
  echo "File $RUBY is not there, aborting."
  exit
fi

source $RUBY

export DROPBOX=~/Dropbox/

git pull
bundle update
git ci Gemfile* -m 'bundle update'
git st
rake push
git push
