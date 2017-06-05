# photos-location
Call Google Places API for retrieve locations using Exif and a directory to search for photos.

## CREATE YOUR API KEY

https://console.developers.google.com/apis/dashboard

### INSTALL
```go
git clone https://github.com/frakev/photos-location.git
cd photos-location/
go build photos.go
```
### LAUNCH
```go
./photos -key {YourApiKey} -directory /home/
```
