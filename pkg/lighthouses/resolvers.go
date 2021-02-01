package lighthouses

import (
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
	"net/url"

	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/lighthouse-client/pkg/filebrowser"
	gitv2 "github.com/jenkins-x/lighthouse-client/pkg/git/v2"
	"github.com/jenkins-x/lighthouse-client/pkg/triggerconfig/inrepo"
	"github.com/pkg/errors"
)

// ResolverOptions the options to create a resolver
type ResolverOptions struct {
	scmhelpers.Factory

	FileBrowser       filebrowser.Interface
	Dir               string
	CatalogOwner      string
	CatalogRepository string
}

// AddFlags adds CLI flags
func (o *ResolverOptions) AddFlags(cmd *cobra.Command) {
	o.Factory.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "The directory to look for the .lighthouse folder")
	cmd.Flags().StringVarP(&o.CatalogOwner, "catalog-owner", "", "jenkins-x", "The github owner for the default catalog")
	cmd.Flags().StringVarP(&o.CatalogRepository, "catalog-repo", "", "jx3-pipeline-catalog", "The github repository name for the default catalog")
}

// CreateResolver creates the resolver from the available options
func (o *ResolverOptions) CreateResolver() (*inrepo.UsesResolver, error) {
	f := o.Factory

	fb := o.FileBrowser

	err := f.FindGitToken()
	if err != nil {
		// ignore missing tokens for now
		log.Logger().Debugf("could not detect git token %s", err.Error())
	}

	if fb == nil {
		gitCloneUser := f.GitUsername
		token := f.GitToken
		if f.GitServerURL == "" {
			f.GitServerURL = "https://github.com"
		}
		var gitServerURL *url.URL
		if f.GitServerURL != "" {
			gitServerURL, err = url.Parse(f.GitServerURL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse git URL %s", f.GitServerURL)
			}

		}

		configureOpts := func(opts *gitv2.ClientFactoryOpts) {
			opts.Token = func() []byte {
				return []byte(token)
			}
			opts.GitUser = func() (name, email string, err error) {
				name = gitCloneUser
				return
			}
			opts.Username = func() (login string, err error) {
				login = gitCloneUser
				return
			}
			if gitServerURL != nil {
				opts.Host = gitServerURL.Host
				opts.Scheme = gitServerURL.Scheme
			}
		}
		gitFactory, err := gitv2.NewClientFactory(configureOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create git factory")
		}
		fb = filebrowser.NewFileBrowserFromGitClient(gitFactory)
	}

	fileBrowsers, err := filebrowser.NewFileBrowsers(f.GitServerURL, fb)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create file browsers")
	}

	return &inrepo.UsesResolver{
		FileBrowsers: fileBrowsers,
		OwnerName:    o.CatalogOwner,
		RepoName:     o.CatalogRepository,
		//Dir:              f.Dir,
		Dir:              "",
		LocalFileResolve: true,
	}, nil
}