# Querying the CTAN with JSON

The CTAN provides mean to access the database and retrieve the information in form of JSON responses. Several entities can be queried.

1. The list of packages can be obtained under the URL `http://www.ctan.org/json/2.0/packages`
1. Extract the `key` from the above list and obtain detailed information about the package using that `key`.
1. The information about a single package can be obtained under the URL like `http://www.ctan.org/json/2.0/pkg/{key}`

Getting detailed information of all packages using the above method is very slow. Therefore, use `oproc` to speed it up.
