/*
	Project: Masomo - https://masomo.cd (ref: https://classroom.google.com/)
	Target: École secondaires (Universities later..)
*/
package masomo

/*
TODO: cmd to add School !!!
TODO: admin: upload CSV to bulk create users via API ???

TODO: "github.com/pkg/errors" !!! for context & error stack

FE: Material Design | PrimeVue | mixture etc..
	- Admin Site
		* manage everything
		* assign admins roles
		* assign courses to teachers
		* manage students & assign them to classes
	- Public Site
		* Teacher Dashboard
		* Student Dashboard
	- avatar: https://github.com/JiriChara/vue-gravatar

------------------------------------ Version X ----------------------------------------
FIXME:Edge-case:
- User belongs to multiple Schools ??? like Teachers ???: 1 User -> diff Teacher per School (creds per School then!!??)
- Student moving to another School ???: same as above + deactivate prev Student??..

TODO: Subscription Model
- Trial Model:
	* TODO (trial expiry dates..)
- Model:
	* when payment made:
		- paymentDueDate = now + 1 Year (variable: Y | M)
		- paymentExtensionDate = paymentDueDate + 1 Month ??
	* else:
		- paymentDueDate reached: exponential notifications until payment is made | paymentExtensionDate
		- paymentExtensionDate reached: deactivate School
	* NAME IT 💰💰💰 (Y & M costs). Notify owners onPriceChange
- cmds:
	* newPayment: - MANUAL...
		- update dates
		- send thank you notification to owner
	* paymentWarning (periodic - exponential): exponential notifications until payment is made | paymentExtensionDate
	* deactivateSchool (daily): deactivate if paymentExtensionDate reached
- if Inactive and user logs in display eg. "School Unavailable!"

 TODO: Calendar

 TODO: CourseWork (M --> M Lessons) - Tutorials & Assignments
	- Physical:
		* content: text | doc
		* response: hard copy (to be physically returned in class)
		* due date
	- Virtual:
		* content: text | doc
		* response: multiple-choice formats (radio | checkboxes) randomized OnInit
		* automatic grading: automatically update Virtual Marks
		* start date:
			- cannot start before start date; should be dictated by CourseContent !!! (grayed out)
			- when setting start date, due date must also be set for Assignments
		* due date:
			- Tutorial: not affected; can be retaken as many times as possible. TODO: show marks history ??
			- Assignment: can be retaken as many times as possible (but grades won't update after due date)
			- show work due soon (2 weeks window) in student home page
		* TODO: Virtual Wallet: (AaaG: Assessment as a Game)
			- Student earns points when they meet criteria
			  set by Teacher based on allocated budget for the Course by Admin
			- Admin allocate budget for each Course every year
			  and reward the `n top richest Students among those with >=65% in final` of each class at the end of the year
			  and reward the best student of each category even more: `YearLevel`, `Department`, `School`

 TODO: Marks
	- Virtual:
		* automatically updated by Virtual Tutorials & Assignments
		* Assignments can be considered for Official Marks by Teacher
	- Official (sync with Physical): to be manually updated by Teacher (TODO: CSV import?)
		* Admin controls when new Marks will be viewable by Students

 TODO: Communication
 	- TODO: Announcements (scheduled) from Admin:
		* School Announcement: all Students + teachers
		* Department Announcement: all Students + teachers of Department
		* Class Announcement: all Students + teachers of Class
 	- TODO: Group Chat + count badge:
 		* Course ChatRoom:
 			- General: all Students + Teacher (admin)
 			- Per Topic OR Group work: all | group of Students + Teacher (admin)
 	- TODO: Notifications:
 		* Announcements
 		*
 	- TODO: Video conferencing (Live courses)

TODO: `Class`
	- copy new class for every year:
		* archive previous year' class (readonly: viewable by related users)
		* copy CourseContents and Tutorials & Assignments (reset start & due dates)

TODO: Progressive Web App: for usage in low network areas !!!
*/
