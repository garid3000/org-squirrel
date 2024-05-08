test_step1:
	emacs -q --batch --eval "(require 'org)" ./test_orgs/step1/step1.org --funcall org-html-export-to-html
	mv ./test_orgs/step1/step1.html ./test_orgs/step1/step1_by_emacs.html
	go run main.go -in ./test_orgs/step1/step1.org -out=./test_orgs/step1/step1_by_org_squirrel.html -type=html
	prettier ./test_orgs/step1/step1_by_emacs.html > ./test_orgs/step1/step1_by_emacs_pretty.html
	prettier ./test_orgs/step1/step1_by_org_squirrel.html > ./test_orgs/step1/step1_by_org_squirrel_pretty.html
