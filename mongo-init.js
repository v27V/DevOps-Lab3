// mongo-init.js
db = db.getSiblingDB('products_db');

db.createUser({
  user: 'user',
  pwd:  'qwerty123',
  roles: [
    {	role: 'dbOwner',
	db: 'products_db',	},
  ],
});
