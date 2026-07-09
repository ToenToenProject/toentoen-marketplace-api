import os
import re

base_dir = "d:/ToenToen/webapi-go"
domain_dir = os.path.join(base_dir, "internal", "domain")

type_to_pkg = {
    'ProfileService': 'account',
    'ProfileRepository': 'account',
    'AuthService': 'auth',
    'AuthRepository': 'auth',
    'BusinessService': 'business',
    'BusinessRepository': 'business',
    'BusinessAdminService': 'admin',
    'BusinessAdminRepository': 'admin',
    'ProductService': 'catalog',
    'ProductRepository': 'catalog',
    'ProductGroupService': 'catalog',
    'ProductGroupRepository': 'catalog',
    'BusinessPostService': 'post',
    'BusinessPostRepository': 'post',
    'AnalyticsService': 'analytics',
    'AnalyticsRepository': 'analytics',
    'DemandService': 'demand',
    'DemandRepository': 'demand',
    'ReviewService': 'review',
    'ReviewRepository': 'review',
    'ProductInteractionService': 'interaction',
    'ProductInteractionRepository': 'interaction',
    'PostInteractionService': 'interaction',
    'PostInteractionRepository': 'interaction',
}

pkg_to_path = {
    'account': 'toentoen-backend/internal/domain/account',
    'auth': 'toentoen-backend/internal/domain/auth',
    'business': 'toentoen-backend/internal/domain/merchant/business',
    'admin': 'toentoen-backend/internal/domain/merchant/admin',
    'catalog': 'toentoen-backend/internal/domain/merchant/catalog',
    'post': 'toentoen-backend/internal/domain/merchant/post',
    'analytics': 'toentoen-backend/internal/domain/merchant/analytics',
    'demand': 'toentoen-backend/internal/domain/customer/demand',
    'review': 'toentoen-backend/internal/domain/customer/review',
    'interaction': 'toentoen-backend/internal/domain/customer/interaction',
}

def fix_file(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Determine current package
    match = re.search(r'^package\s+([a-z0-9_]+)', content, flags=re.MULTILINE)
    if not match: return
    current_pkg = match.group(1)

    # Find all usages of services.XYZ or repositories.XYZ
    used_types = re.findall(r'(?:services|repositories)\.([A-Za-z0-9_]+)', content)
    
    # We also need to strip old imports
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/domain/handlers_old".*$\n', '', content)
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/domain/services_old".*$\n', '', content)
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/domain/repositories_old".*$\n', '', content)
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/handlers".*$\n', '', content)
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/services".*$\n', '', content)
    content = re.sub(r'(?m)^\s*"toentoen-backend/internal/repositories".*$\n', '', content)

    needed_imports = set()

    # Replace usages
    for t in set(used_types):
        if t in type_to_pkg:
            target_pkg = type_to_pkg[t]
            if target_pkg == current_pkg:
                # Same package, remove prefix
                content = re.sub(r'(services|repositories)\.' + t + r'\b', t, content)
            else:
                # Different package, change prefix to target_pkg and add to imports
                content = re.sub(r'(services|repositories)\.' + t + r'\b', target_pkg + '.' + t, content)
                needed_imports.add(pkg_to_path[target_pkg])
        else:
            # If not in our map (maybe misspelled or internal struct), just strip prefix if it's the same package?
            # Actually we shouldn't have other types exported. Just strip it.
            content = re.sub(r'(services|repositories)\.' + t + r'\b', t, content)

    if needed_imports:
        imports_str = '\n'.join([f'\t"{i}"' for i in needed_imports])
        # Insert into the import block
        content = re.sub(r'(import \()', r'\1\n' + imports_str, content)

    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)


for root, _, files in os.walk(domain_dir):
    for f in files:
        if f.endswith('.go'):
            fix_file(os.path.join(root, f))

# Also we need to fix main.go
main_file = os.path.join(base_dir, "cmd", "api", "main.go")
with open(main_file, 'r', encoding='utf-8') as f:
    main_content = f.read()

# Replace old imports with new domain imports
main_imports = [
    '\t"toentoen-backend/internal/domain/account"',
    '\t"toentoen-backend/internal/domain/auth"',
    '\t"toentoen-backend/internal/domain/merchant/business"',
    '\t"toentoen-backend/internal/domain/merchant/admin"',
    '\t"toentoen-backend/internal/domain/merchant/catalog"',
    '\t"toentoen-backend/internal/domain/merchant/post"',
    '\t"toentoen-backend/internal/domain/merchant/analytics"',
    '\t"toentoen-backend/internal/domain/customer/demand"',
    '\t"toentoen-backend/internal/domain/customer/review"',
    '\t"toentoen-backend/internal/domain/customer/interaction"',
]
main_content = re.sub(r'(?m)^\s*"toentoen-backend/internal/handlers".*$\n', '', main_content)
main_content = re.sub(r'(?m)^\s*"toentoen-backend/internal/services".*$\n', '', main_content)
main_content = re.sub(r'(?m)^\s*"toentoen-backend/internal/repositories".*$\n', '', main_content)

main_content = re.sub(r'(import \()', r'\1\n' + '\n'.join(main_imports), main_content)

# Update instantiations in main.go
# e.g., repositories.NewAuthRepository -> auth.NewAuthRepository
for t, target_pkg in type_to_pkg.items():
    main_content = re.sub(r'repositories\.(New' + t + r')', target_pkg + r'.\1', main_content)
    main_content = re.sub(r'services\.(New' + t + r')', target_pkg + r'.\1', main_content)
    # Handlers follow a pattern like NewAuthHandler
    handler_name = t.replace('Service', 'Handler')
    main_content = re.sub(r'handlers\.(New' + handler_name + r')', target_pkg + r'.\1', main_content)
    
# also authHandler.Logout -> auth.Logout if it was a generic function in handlers
main_content = main_content.replace('handlers.Logout', 'auth.Logout')

with open(main_file, 'w', encoding='utf-8') as f:
    f.write(main_content)

print("Cross imports fixed.")
