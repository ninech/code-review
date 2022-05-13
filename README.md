# Code Review Exercise

## Setup

To properly review this exercise (and run the tests), you will need to have `go`
installed.

Further, the tests require some additional assets, which can be downloaded by
calling the script `setup.sh` within the `test_assets` directory. This script
should create a directory `test_assets/kubebuilder/bin`, containing an `etcd`,
`kubectl` and a `kube-apiserver` binary.

The tests can be triggered through `go test`, and should succeed. Let us know if
that should not be the case! This has been tested with Go 1.17, though we expect
other versions to work as well.

## Reviewing the Code

We recommend checking the code out locally in order to get a full picture of the
code under review. Please note that this exercise is not just about the code
changes, but also about the already existing code. Try to understand the current
system and the extension that is proposed in this PR.

Another recommendation: be nit-picky! We won't judge you for having opinions
about how code should look like, and are interested in seeing what you consider
idiomatic Go code and bad and good practice.

We're open to hearing your thoughts about anything in this project, even if you
find typos in this README :-)

You may take as much or as little time to do this exercise as you wish and have.

## Limitations

In `controller/oidc.go` you can find stubs of what would be a client to get
information about an OIDC client. The actual implementation is not relevant to
this exercise, but try to think about what would happen if this was implemented.
What would change, for example when running tests? What would be required to
make an implementation work? Are there limitations to the current stub?

## Next Steps

Once we've talked through your review of the code, we will ask you to do some
live coding. You may prepare some thoughts on what you think would be sensible
to fix, clean up or add. This will be similar to a pair programming exercise,
and we expect this to be interactive with the interviewers.

## Further Information

You may find further information about how to write Kubernetes reconcilers in
the [kubebuilder book](https://book.kubebuilder.io/introduction.html) (we don't
expect you to have read that and will not ask you questions out of it!). However
it might help understand the existing code.
