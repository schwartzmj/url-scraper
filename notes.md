# Notes

- We may want to keep track of redirects we've followed and mark every redirect as visited/handled/whatever?
  - Since we know where they go to? Save each redirect as a page visited? Save in separate map? I don't know.
- We do not ToLower or ToUpper the URLs we're checking, since many websites are case-sensitive. Should we allow a flag to toggle this on/off?