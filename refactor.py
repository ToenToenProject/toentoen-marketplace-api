import os
import shutil
import re

base_dir = "d:/ToenToen/webapi-go"
internal_dir = os.path.join(base_dir, "internal")

# Mapping of file source to destination and package name
moves = [
    # Auth
    ("handlers/auth_handler.go", "domain/auth/handler.go", "auth"),
    ("handlers/logout.go", "domain/auth/logout.go", "auth"),
    ("services/auth_service.go", "domain/auth/service.go", "auth"),
    ("repositories/auth_repository.go", "domain/auth/repository.go", "auth"),
    
    # Account
    ("handlers/profile_handler.go", "domain/account/handler.go", "account"),
    ("services/profile_service.go", "domain/account/service.go", "account"),
    ("repositories/profile_repository.go", "domain/account/repository.go", "account"),

    # Merchant - Business
    ("handlers/business_handler.go", "domain/merchant/business/handler.go", "business"),
    ("services/business_service.go", "domain/merchant/business/service.go", "business"),
    ("repositories/business_repository.go", "domain/merchant/business/repository.go", "business"),

    # Merchant - Admin
    ("handlers/business_admin_handler.go", "domain/merchant/admin/handler.go", "admin"),
    ("services/business_admin_service.go", "domain/merchant/admin/service.go", "admin"),
    ("repositories/business_admin_repository.go", "domain/merchant/admin/repository.go", "admin"),

    # Merchant - Catalog
    ("handlers/product_handler.go", "domain/merchant/catalog/product_handler.go", "catalog"),
    ("services/product_service.go", "domain/merchant/catalog/product_service.go", "catalog"),
    ("repositories/product_repository.go", "domain/merchant/catalog/product_repository.go", "catalog"),
    ("handlers/product_group_handler.go", "domain/merchant/catalog/product_group_handler.go", "catalog"),
    ("services/product_group_service.go", "domain/merchant/catalog/product_group_service.go", "catalog"),
    ("repositories/product_group_repository.go", "domain/merchant/catalog/product_group_repository.go", "catalog"),

    # Merchant - Post
    ("handlers/business_post_handler.go", "domain/merchant/post/handler.go", "post"),
    ("services/business_post_service.go", "domain/merchant/post/service.go", "post"),
    ("repositories/business_post_repository.go", "domain/merchant/post/repository.go", "post"),

    # Merchant - Analytics
    ("handlers/merchant_analytics_handler.go", "domain/merchant/analytics/handler.go", "analytics"),
    ("services/merchant_analytics_service.go", "domain/merchant/analytics/service.go", "analytics"),
    ("repositories/merchant_analytics_repository.go", "domain/merchant/analytics/repository.go", "analytics"),

    # Customer - Demand
    ("handlers/demand_handler.go", "domain/customer/demand/handler.go", "demand"),
    ("services/demand_service.go", "domain/customer/demand/service.go", "demand"),
    ("repositories/demand_repository.go", "domain/customer/demand/repository.go", "demand"),

    # Customer - Review
    ("handlers/review_handler.go", "domain/customer/review/handler.go", "review"),
    ("services/review_service.go", "domain/customer/review/service.go", "review"),
    ("repositories/review_repository.go", "domain/customer/review/repository.go", "review"),

    # Customer - Interaction
    ("handlers/product_interaction_handler.go", "domain/customer/interaction/product_interaction_handler.go", "interaction"),
    ("services/product_interaction_service.go", "domain/customer/interaction/product_interaction_service.go", "interaction"),
    ("repositories/product_interaction_repository.go", "domain/customer/interaction/product_interaction_repository.go", "interaction"),
    ("handlers/business_post_interaction_handler.go", "domain/customer/interaction/business_post_interaction_handler.go", "interaction"),
    ("services/business_post_interaction_service.go", "domain/customer/interaction/business_post_interaction_service.go", "interaction"),
    ("repositories/business_post_interaction_repository.go", "domain/customer/interaction/business_post_interaction_repository.go", "interaction"),
]

# Shared
shared_moves = []
for m_file in os.listdir(os.path.join(internal_dir, "models")):
    shared_moves.append((f"models/{m_file}", f"domain/shared/models/{m_file}", "models"))
for m_file in os.listdir(os.path.join(internal_dir, "middlewares")):
    shared_moves.append((f"middlewares/{m_file}", f"domain/shared/middlewares/{m_file}", "middlewares"))

# Perform moving
all_moves = moves + shared_moves
for src, dst, pkg in all_moves:
    src_path = os.path.join(internal_dir, src)
    dst_path = os.path.join(internal_dir, dst)
    if not os.path.exists(src_path):
        print(f"File {src_path} not found")
        continue
    os.makedirs(os.path.dirname(dst_path), exist_ok=True)
    shutil.move(src_path, dst_path)

# Update packages and imports in all go files in internal/domain
import_replacements = {
    '"toentoen-backend/internal/models"': '"toentoen-backend/internal/domain/shared/models"',
    '"toentoen-backend/internal/middlewares"': '"toentoen-backend/internal/domain/shared/middlewares"',
    # Instead of just services_old, we'll need to do some more complex replacements later in main.go
}

def update_imports(file_path, pkg_name):
    if not os.path.exists(file_path): return
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Replace package declaration
    if pkg_name:
        content = re.sub(r'^package\s+(handlers|services|repositories|models|middlewares)', f'package {pkg_name}', content, flags=re.MULTILINE)
    
    # Replace imports
    for old, new in import_replacements.items():
        content = content.replace(old, new)
        
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)

for root, _, files in os.walk(os.path.join(internal_dir, "domain")):
    for file in files:
        if file.endswith('.go'):
            pkg = None
            for src, dst, p in all_moves:
                if dst.endswith(file) and os.path.normpath(os.path.join(internal_dir, dst)) == os.path.normpath(os.path.join(root, file)):
                    pkg = p
                    break
            if not pkg:
                if 'models' in root: pkg = 'models'
                if 'middlewares' in root: pkg = 'middlewares'
            update_imports(os.path.join(root, file), pkg)

print("Moved files and updated package names.")
