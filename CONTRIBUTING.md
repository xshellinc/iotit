# How to contribute

Community submissions are essential for making `iotit` the best tool for flashing single board computers(SBCs). Without your help we would not be able to keep up with all the developments in this area.

## Getting Started

* Make sure you have a [GitHub account](https://github.com/signup/free)
* Submit a ticket for your issue, if one does not already exist.
  * Clearly describe the issue including steps to reproduce if it is a bug.
  * Make sure you fill in the earliest version that you know has the issue.
* Fork the repository on GitHub

## Making Changes

* Create a topic branch from where you want to base your work.
  * This is usually the master branch.
  * Only target release branches if you are certain your fix must be on that
    branch.
  * To quickly create a topic branch based on master; `git checkout -b
    fix/master/my_contribution master`. Please avoid working directly on the
    `master` branch.
* Make commits of logical units.
* Check for unnecessary whitespace with `git diff --check` before committing.
* Make sure your commit messages are in the proper format, see [here](https://chris.beams.io/posts/git-commit/) for the format we use.
* Make sure you have added the necessary tests for your changes.
* Run _all_ the tests to assure nothing else was accidentally broken.

## Submitting Changes

* Push your changes to a topic branch in your fork of the repository.
* Submit a pull request to the repository in the xshell organization.
* Update the github issue to mark that you have submitted code and are ready for it to be reviewed.
  * Include a link to the pull request in the issue.
  * If you are pushing new feature please also update the README to explain clearly what your new feature is and how it works.
* We will review the pull request and provide feedback. Either it will be accepted or we will ask you to make some changes if we find some problem.
