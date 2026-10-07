{{- /*
  Hextra's page.markdown.md, with ../ links made absolute. The Markdown copy
  is served at docs/<page>.md, one level above the HTML at docs/<page>/, so a
  relative ../nfe/ would point outside docs/. Both now resolve to docs/nfe/.
*/ -}}
{{- $parent := path.Dir (strings.TrimSuffix "/" .RelPermalink) -}}
{{- .Title | replaceRE "\n" " " | printf "# %s" }}
{{ replace .RawContent "](../" (printf "](%s/" $parent) }}
