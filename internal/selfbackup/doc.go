// Package selfbackup periodically dumps the control plane's PostgreSQL database
// and stores the result in a Restic repository. It operates independently of the
// catalog so that a database failure does not prevent the backup from running.
package selfbackup
