# Sites Directory

Copy or symlink site directories here. The structure of the site directory must be:

```
sitename.tld
|-build
  |-main    # main executable if the site is dynamic. Takes one argument (FCGI socket path)
  |-index.html  # HTML file(s) and other static file(s) if the site is static.
```
