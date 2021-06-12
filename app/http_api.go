package app

import "net/http"

func (v *vxdb) apiBackup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/octet-stream")
	w.Header().Set("content-disposition", "attachment; filename=backup.blob")

	if _, err := v.db.Backup(w, 0); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (v *vxdb) apiRestore(w http.ResponseWriter, r *http.Request) {
	if err := v.db.Load(r.Body, 256); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
