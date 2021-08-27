# Monzo Scraper

## Running

```
mscraper . -url https://makalila.dev/ -depth 1 -verbose
```
- url is the starting url.
- depth dictates how deep to follow links, default is 1.
    - depth 0 will only fetch the starting url.
    - depth 1 will fetch the starting url, and follow all links in the starting url page.
- verbose determines whether to print out a url and the links within it, default in false.
- With verbose on, the output will be similar to (both visited links the the links within them are printed):
```
- https://makalila.dev/
   - https://makalila.dev/
   - https://makalila.dev/blog
   - https://makalila.dev/blog/python-socket-communication/
   - https://makalila.dev/blog/json-the-python-way/
   - https://makalila.dev/blog/python-dictionaries/
- https://makalila.dev/blog/python-dictionaries/
   - https://makalila.dev/
   - https://makalila.dev/categories/programming/
   - https://makalila.dev/tags/python/
   - https://makalila.dev/tags/json/
- https://makalila.dev/blog/json-the-python-way/
   - https://makalila.dev/
   - https://makalila.dev/categories/programming/
   - https://makalila.dev/tags/python/
   - https://makalila.dev/tags/json/
   - https://makalila.dev/tags/data/
```
- With verbose off (deafult),the output will be similar to (only the visited links are printed):
```
- https://makalila.dev/
- https://makalila.dev/blog/python-dictionaries/
- https://makalila.dev/blog/json-the-python-way/
- https://makalila.dev/blog/python-socket-communication/
- https://makalila.dev/blog
```